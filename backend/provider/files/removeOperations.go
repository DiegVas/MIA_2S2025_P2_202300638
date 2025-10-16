package files

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"main/provider/common"
	"main/provider/mounted"
	"main/provider/reports"
	"main/structs"
	"main/utils/journaling"
	"os"
	"path/filepath"
)

// RemoveFileOrDirectory elimina un archivo o directorio y todo su contenido
func RemoveFileOrDirectory(params *structs.RemoveParams) error {
	// Obtener la sesión actual
	session := structs.GetSession()
	if !session.Active {
		return errors.New("debe iniciar sesión para usar el comando REMOVE")
	}

	// Obtener la partición montada
	mountManager := mounted.GetMountManager()
	partition, found := mountManager.GetMountedPartition(session.ID)
	if !found {
		return errors.New("no hay ninguna partición montada para esta sesión")
	}

	// Leer el superbloque usando common
	superblock, err := common.ReadSuperblock(partition.Path, session.ID)
	if err != nil {
		return fmt.Errorf("error leyendo superbloque: %v", err)
	}

	// Abrir el archivo del disco
	file, err := os.OpenFile(partition.Path, os.O_RDWR, 0644)
	if err != nil {
		return fmt.Errorf("error abriendo el disco: %v", err)
	}
	defer file.Close()

	// Buscar el inodo del archivo/directorio usando reports
	targetInodeNum, err := reports.FindInodeByPath(file, params.Path, superblock)
	if err != nil {
		return fmt.Errorf("archivo o directorio no encontrado: %v", err)
	}

	// Leer el inodo objetivo
	targetInode, err := reports.ReadInodeAt(file, targetInodeNum, superblock.S_inode_start)
	if err != nil {
		return fmt.Errorf("error leyendo inodo: %v", err)
	}

	// Verificar permisos de escritura en el directorio padre
	parentPath := filepath.Dir(params.Path)
	if parentPath == "." {
		parentPath = "/"
	}

	parentInodeNum, err := reports.FindInodeByPath(file, parentPath, superblock)
	if err != nil {
		return fmt.Errorf("directorio padre no encontrado: %v", err)
	}

	if !hasWritePermission(file, parentInodeNum, session, superblock) {
		return errors.New("no tiene permisos de escritura en el directorio padre")
	}

	// Si es un directorio, verificar permisos recursivamente
	if targetInode.I_type == 1 { // Es directorio
		canDelete, err := canDeleteDirectoryRecursively(file, targetInodeNum, session, superblock)
		if err != nil {
			return fmt.Errorf("error verificando permisos: %v", err)
		}
		if !canDelete {
			return errors.New("no tiene permisos suficientes para eliminar el directorio y todo su contenido")
		}
	} else { // Es archivo
		if !hasWritePermission(file, targetInodeNum, session, superblock) {
			return errors.New("no tiene permisos de escritura sobre el archivo")
		}
	}

	// Proceder con la eliminación
	err = removeInodeRecursively(file, targetInodeNum, superblock)
	if err != nil {
		return fmt.Errorf("error eliminando: %v", err)
	}

	// Remover la entrada del directorio padre
	err = removeFromParentDirectory(file, parentInodeNum, filepath.Base(params.Path), superblock)
	if err != nil {
		return fmt.Errorf("error removiendo entrada del directorio padre: %v", err)
	}

	// Actualizar el superbloque
	err = updateSuperblock(file, superblock, partition.Path, session.ID)
	if err != nil {
		return fmt.Errorf("error actualizando superbloque: %v", err)
	}

	// Registrar la eliminación en el journal EXT3
	journaling.LogEXT3Operation(session.ID, "REMOVE_FILE_OR_DIRECTORY", params.Path,
		fmt.Sprintf("Archivo/directorio '%s' eliminado exitosamente", params.Path))

	return nil
}

// canDeleteDirectoryRecursively verifica si se puede eliminar un directorio y todo su contenido
func canDeleteDirectoryRecursively(file *os.File, dirInodeNum int32, session *structs.Session, sb *structs.Superblock) (bool, error) {
	// Verificar permisos en el directorio actual
	if !hasWritePermission(file, dirInodeNum, session, sb) {
		return false, nil
	}

	// Leer el contenido del directorio usando reports
	dirInode, err := reports.ReadInodeAt(file, dirInodeNum, sb.S_inode_start)
	if err != nil {
		return false, err
	}

	entries, err := reports.GetDirectoryEntries(file, dirInode, sb, dirInodeNum)
	if err != nil {
		return false, err
	}

	// Verificar permisos en cada entrada
	for _, entry := range entries {
		if entry.Name == "." || entry.Name == ".." {
			continue
		}

		if entry.InodeNum == 0 {
			continue
		}

		// Leer el inodo de la entrada
		entryInode, err := reports.ReadInodeAt(file, entry.InodeNum, sb.S_inode_start)
		if err != nil {
			continue
		}

		// Si es directorio, verificar recursivamente
		if entryInode.I_type == 1 {
			canDelete, err := canDeleteDirectoryRecursively(file, entry.InodeNum, session, sb)
			if err != nil || !canDelete {
				return false, err
			}
		} else {
			// Si es archivo, verificar permisos de escritura
			if !hasWritePermission(file, entry.InodeNum, session, sb) {
				return false, nil
			}
		}
	}

	return true, nil
}

// removeInodeRecursively elimina un inodo y todo su contenido recursivamente
func removeInodeRecursively(file *os.File, inodeNum int32, sb *structs.Superblock) error {

	// Leer el inodo
	inode, err := reports.ReadInodeAt(file, inodeNum, sb.S_inode_start)
	if err != nil {
		return err
	}

	// Si es directorio, eliminar contenido recursivamente
	if inode.I_type == 1 {
		entries, err := reports.GetDirectoryEntries(file, inode, sb, inodeNum)
		if err != nil {
			return err
		}

		for _, entry := range entries {
			if entry.Name == "." || entry.Name == ".." {
				continue
			}

			if entry.InodeNum == 0 {
				continue
			}

			err = removeInodeRecursively(file, entry.InodeNum, sb)
			if err != nil {
				return err
			}
		}
	}

	// Liberar bloques del inodo

	err = freeInodeBlocks(file, inode, sb)
	if err != nil {
		return err
	}

	// Marcar inodo como libre

	err = markInodeAsFree(file, inodeNum, sb)
	if err != nil {
		return err
	}

	// Incrementar contador de inodos libres
	sb.S_free_inodes_count++

	return nil
}

// freeInodeBlocks libera todos los bloques utilizados por un inodo
func freeInodeBlocks(file *os.File, inode *structs.Inode, sb *structs.Superblock) error {
	// Liberar bloques directos
	for i := 0; i < 12; i++ {
		if inode.I_block[i] != 0 {
			err := markBlockAsFree(file, inode.I_block[i], sb)
			if err != nil {
				return err
			}
			sb.S_free_blocks_count++
		}
	}

	// Liberar bloque indirecto (si existe)
	if inode.I_block[12] != 0 {
		err := freeIndirectBlock(file, inode.I_block[12], sb)
		if err != nil {
			return err
		}
	}

	// Liberar bloque doblemente indirecto (si existe)
	if inode.I_block[13] != 0 {
		err := freeDoubleIndirectBlock(file, inode.I_block[13], sb)
		if err != nil {
			return err
		}
	}

	// Liberar bloque triplemente indirecto (si existe)
	if inode.I_block[14] != 0 {
		err := freeTripleIndirectBlock(file, inode.I_block[14], sb)
		if err != nil {
			return err
		}
	}

	return nil
}

// Funciones auxiliares para liberar bloques indirectos
func freeIndirectBlock(file *os.File, blockNum int32, sb *structs.Superblock) error {
	blockPos := sb.S_block_start + (blockNum * structs.BLOCK_SIZE)

	// Leer bloque indirecto
	blockData := make([]byte, structs.BLOCK_SIZE)
	_, err := file.ReadAt(blockData, int64(blockPos))
	if err != nil {
		return err
	}

	// Procesar punteros en el bloque
	numPointers := int32(structs.BLOCK_SIZE / 4)
	for i := int32(0); i < numPointers; i++ {
		pointer := binary.LittleEndian.Uint32(blockData[i*4:])
		if pointer != 0 {
			err = markBlockAsFree(file, int32(pointer), sb)
			if err != nil {
				return err
			}
			sb.S_free_blocks_count++
		}
	}

	// Liberar el bloque indirecto
	err = markBlockAsFree(file, blockNum, sb)
	if err != nil {
		return err
	}
	sb.S_free_blocks_count++

	return nil
}

func freeDoubleIndirectBlock(file *os.File, blockNum int32, sb *structs.Superblock) error {
	blockPos := sb.S_block_start + (blockNum * structs.BLOCK_SIZE)

	// Leer bloque doblemente indirecto
	blockData := make([]byte, structs.BLOCK_SIZE)
	_, err := file.ReadAt(blockData, int64(blockPos))
	if err != nil {
		return err
	}

	// Procesar punteros a bloques indirectos
	numPointers := int32(structs.BLOCK_SIZE / 4)
	for i := int32(0); i < numPointers; i++ {
		pointer := binary.LittleEndian.Uint32(blockData[i*4:])
		if pointer != 0 {
			err = freeIndirectBlock(file, int32(pointer), sb)
			if err != nil {
				return err
			}
		}
	}

	// Liberar el bloque doblemente indirecto
	err = markBlockAsFree(file, blockNum, sb)
	if err != nil {
		return err
	}
	sb.S_free_blocks_count++

	return nil
}

func freeTripleIndirectBlock(file *os.File, blockNum int32, sb *structs.Superblock) error {
	blockPos := sb.S_block_start + (blockNum * structs.BLOCK_SIZE)

	// Leer bloque triplemente indirecto
	blockData := make([]byte, structs.BLOCK_SIZE)
	_, err := file.ReadAt(blockData, int64(blockPos))
	if err != nil {
		return err
	}

	// Procesar punteros a bloques doblemente indirectos
	numPointers := int32(structs.BLOCK_SIZE / 4)
	for i := int32(0); i < numPointers; i++ {
		pointer := binary.LittleEndian.Uint32(blockData[i*4:])
		if pointer != 0 {
			err = freeDoubleIndirectBlock(file, int32(pointer), sb)
			if err != nil {
				return err
			}
		}
	}

	// Liberar el bloque triplemente indirecto
	err = markBlockAsFree(file, blockNum, sb)
	if err != nil {
		return err
	}
	sb.S_free_blocks_count++

	return nil
}

// markInodeAsFree marca un inodo como libre en el bitmap
func markInodeAsFree(file *os.File, inodeNum int32, sb *structs.Superblock) error {
	bmInodePos := sb.S_bm_inode_start
	writePos := bmInodePos + inodeNum

	_, err := file.WriteAt([]byte{0}, int64(writePos))
	if err != nil {

	}
	return err
}

// markBlockAsFree marca un bloque como libre en el bitmap
func markBlockAsFree(file *os.File, blockIndex int32, sb *structs.Superblock) error {
	bmBlockPos := sb.S_bm_block_start
	writePos := bmBlockPos + blockIndex

	_, err := file.WriteAt([]byte{0}, int64(writePos))
	if err != nil {
		fmt.Printf("DEBUG MARK_BLOCK_FREE: ERROR escribiendo en posición %d: %v\n", writePos, err)
	}
	return err
}

// removeFromParentDirectory remueve una entrada del directorio padre
func removeFromParentDirectory(file *os.File, parentInodeNum int32, filename string, sb *structs.Superblock) error {
	// Leer el inodo del directorio padre
	parentInode, err := reports.ReadInodeAt(file, parentInodeNum, sb.S_inode_start)
	if err != nil {
		return fmt.Errorf("error leyendo inodo padre: %v", err)
	}

	// Buscar en TODOS los bloques del directorio, incluyendo bloques indirectos
	found := false

	// Primero buscar en bloques directos
	for i := 0; i < 12; i++ {
		if parentInode.I_block[i] == 0 {
			continue // Saltar bloques vacíos pero continuar con los siguientes
		}

		// Usar la misma lógica que GetDirectoryEntries para calcular la posición
		var blockPos int32
		if sb.S_block_start == 0 {
			// Calcular: inode_start + (inodes_count * inode_size)
			blockPos = sb.S_inode_start + (sb.S_inodes_count * structs.INODE_SIZE)
			blockPos += (parentInode.I_block[i] * structs.BLOCK_SIZE)
		} else {
			blockPos = sb.S_block_start + (parentInode.I_block[i] * structs.BLOCK_SIZE)
		}

		if searchAndRemoveInBlock(file, int64(blockPos), filename) {
			found = true
			break
		}
	}

	// Si no se encontró en bloques directos, buscar en bloque indirecto
	if !found && parentInode.I_block[12] != 0 {

		found = searchAndRemoveInIndirectBlock(file, parentInode.I_block[12], filename, sb)
	}

	if !found {
		return fmt.Errorf("entrada '%s' no encontrada en directorio padre", filename)
	}

	return nil
}

// searchAndRemoveInBlock busca y elimina una entrada en un bloque específico
func searchAndRemoveInBlock(file *os.File, blockPos int64, filename string) bool {
	fmt.Printf("DEBUG REMOVE: Buscando '%s' en bloque en posición %d\n", filename, blockPos)
	var folderBlock structs.FolderBlock
	blockBytes := make([]byte, structs.BLOCK_SIZE)
	_, err := file.ReadAt(blockBytes, blockPos)
	if err != nil {
		fmt.Printf("DEBUG REMOVE: Error leyendo bloque en pos %d: %v\n", blockPos, err)
		return false
	}

	buffer := bytes.NewReader(blockBytes)
	binary.Read(buffer, binary.LittleEndian, &folderBlock)

	// Buscar la entrada a remover
	for j := 0; j < 4; j++ {
		entryName := string(bytes.TrimRight(folderBlock.B_content[j].B_name[:], "\x00"))

		if entryName == filename && folderBlock.B_content[j].B_inodo != 0 {

			// Limpiar la entrada
			folderBlock.B_content[j].B_name = [12]byte{}
			folderBlock.B_content[j].B_inodo = 0

			// Escribir el bloque actualizado
			var buf bytes.Buffer
			binary.Write(&buf, binary.LittleEndian, &folderBlock)
			_, err = file.WriteAt(buf.Bytes(), blockPos)
			if err != nil {
				fmt.Printf("DEBUG REMOVE: Error escribiendo bloque: %v\n", err)
				return false
			}

			return true
		}
	}
	return false
}

// searchAndRemoveInIndirectBlock busca y elimina en bloque indirecto
func searchAndRemoveInIndirectBlock(file *os.File, indirectBlockNum int32, filename string, sb *structs.Superblock) bool {
	blockPos := sb.S_block_start + (indirectBlockNum * structs.BLOCK_SIZE)

	// Leer bloque indirecto
	blockData := make([]byte, structs.BLOCK_SIZE)
	_, err := file.ReadAt(blockData, int64(blockPos))
	if err != nil {
		fmt.Printf("DEBUG REMOVE: Error leyendo bloque indirecto: %v\n", err)
		return false
	}

	// Procesar punteros en el bloque
	numPointers := int32(structs.BLOCK_SIZE / 4)
	for i := int32(0); i < numPointers; i++ {
		pointer := binary.LittleEndian.Uint32(blockData[i*4:])
		if pointer != 0 {
			dataBlockPos := sb.S_block_start + (int32(pointer) * structs.BLOCK_SIZE)

			if searchAndRemoveInBlock(file, int64(dataBlockPos), filename) {
				return true
			}
		}
	}

	return false
}

func hasWritePermission(file *os.File, inodeNum int32, session *structs.Session, sb *structs.Superblock) bool {
	// Leer el inodo
	inode, err := reports.ReadInodeAt(file, inodeNum, sb.S_inode_start)
	if err != nil {
		return false
	}

	// Si es root, tiene todos los permisos
	if session.User == "root" {
		return true
	}

	// Obtener información del usuario actual (simplificado)
	// En un sistema real, aquí se consultaría el archivo users.txt

	// Verificar permisos de escritura
	perms := inode.I_perm

	// Por simplicidad, permitir escritura si el usuario es el propietario
	// o si tiene permisos de escritura globales
	return (perms & 0222) != 0 // Permisos de escritura básicos
}

func updateSuperblock(file *os.File, sb *structs.Superblock, diskPath, partitionID string) error {
	// Obtener la partición montada para conocer la posición de inicio
	mountManager := mounted.GetMountManager()
	partition, found := mountManager.GetMountedPartition(partitionID)
	if !found {
		return errors.New("partición no encontrada")
	}

	// Leer el MBR para obtener la información de la partición
	mbr, err := common.ReadMBRFromDisk(partition.Path)
	if err != nil {
		return fmt.Errorf("error leyendo MBR: %v", err)
	}

	// Buscar la partición en el MBR
	var startPos int32
	found = false
	for _, p := range mbr.Partitions {
		partName := string(bytes.TrimRight(p.Part_name[:], "\x00"))
		if partName == partition.Name {
			startPos = p.Part_start
			found = true
			break
		}
	}

	if !found {
		return errors.New("partición no encontrada en MBR")
	}

	// Escribir el superbloque actualizado
	var buf bytes.Buffer
	err = binary.Write(&buf, binary.LittleEndian, sb)
	if err != nil {
		return err
	}

	_, err = file.WriteAt(buf.Bytes(), int64(startPos))
	return err
}
