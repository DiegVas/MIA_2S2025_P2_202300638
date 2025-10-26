package files

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	diskio "main/diskManager"
	"main/provider/common"
	"main/provider/mounted"
	"main/provider/users"
	"main/structs"
	"main/utils/journaling"
)

// EditFile edita el contenido de un archivo existente
func EditFile(params *structs.EditParams) error {
	// Verificar que haya una sesión activa
	session := structs.GetSession()
	if !session.Active {
		return errors.New("debe iniciar sesión para usar el comando EDIT")
	}

	// Obtener la partición montada
	mountManager := mounted.GetMountManager()
	partition, found := mountManager.GetMountedPartition(session.ID)
	if !found {
		return errors.New("no hay ninguna partición montada para esta sesión")
	}

	// Leer el superbloque
	superblock, err := common.ReadSuperblock(partition.Path, session.ID)
	if err != nil {
		return fmt.Errorf("error leyendo superbloque: %v", err)
	}

	// Leer el MBR del disco
	mbr, err := diskio.ReadMBR(partition.Path)
	if err != nil {
		return fmt.Errorf("error al leer el MBR del disco: %v", err)
	}

	// Buscar la partición montada
	var targetPartition structs.PARTITION

	var partitionFound bool
	for _, p := range mbr.Partitions {
		partName := strings.TrimRight(string(p.Part_name[:]), "\x00")
		if partName == partition.Name {
			targetPartition = p
			partitionFound = true
			break
		}
	}

	if !partitionFound {
		return fmt.Errorf("la partición '%s' no se encontró en el disco", partition.Name)
	}

	// Calcular las posiciones de las estructuras del sistema de archivos (igual que renameFile)
	startPosition := targetPartition.Part_start
	inodeCount := int32(float64(targetPartition.Part_size) * 0.05 / float64(structs.INODE_SIZE))
	blockCount := (targetPartition.Part_size - inodeCount*structs.INODE_SIZE) / structs.BLOCK_SIZE

	bmInodePos := startPosition + structs.SUPERBLOCK_SIZE
	bmBlockPos := bmInodePos + inodeCount

	// Para blockPos, usar el superbloque (como renameFile)
	var blockPos int32
	if superblock.S_block_start == 0 {
		blockPos = superblock.S_inode_start + (superblock.S_inodes_count * structs.INODE_SIZE)
	} else {
		blockPos = superblock.S_block_start
	}

	// Abrir el archivo del disco para escritura
	file, err := os.OpenFile(partition.Path, os.O_RDWR, 0666)
	if err != nil {
		return fmt.Errorf("error al abrir el disco: %v", err)
	}
	defer file.Close()

	// Buscar el inodo del archivo a editar
	// IMPORTANTE: Usar superblock.S_inode_start para búsqueda
	inodeNum, err := common.FindFileInodeByPath(file, params.Path, superblock, superblock.S_inode_start, blockPos)
	if err != nil {
		return fmt.Errorf("archivo no encontrado: %v", err)
	}

	// Leer el inodo del archivo usando S_inode_start del superbloque
	inodeBytes := make([]byte, structs.INODE_SIZE)
	inodeOffset := superblock.S_inode_start + (inodeNum * structs.INODE_SIZE)
	_, err = file.ReadAt(inodeBytes, int64(inodeOffset))
	if err != nil {
		return fmt.Errorf("error al leer el inodo del archivo: %v", err)
	}

	// Deserializar el inodo
	var inode structs.Inode
	buffer := bytes.NewReader(inodeBytes)
	binary.Read(buffer, binary.LittleEndian, &inode)

	// Verificar que es un archivo (no un directorio)
	if inode.I_type != 0 {
		return errors.New("el path especificado no es un archivo")
	}

	// Obtener información del usuario actual
	currentUser, err := users.GetCurrentUserInfo(session, partition)
	if err != nil {
		return fmt.Errorf("error al obtener información del usuario: %v", err)
	}

	// Verificar permisos de lectura y escritura
	if !hasReadAndWritePermission(inode, currentUser) {
		return errors.New("no tiene permisos de lectura y escritura para este archivo")
	}

	// Leer el contenido del archivo local que contiene el nuevo contenido
	newContent, err := ioutil.ReadFile(params.Contenido)
	if err != nil {
		return fmt.Errorf("error al leer el archivo local '%s': %v", params.Contenido, err)
	}

	newContentStr := string(newContent)
	newSize := int32(len(newContentStr))

	fmt.Printf("DEBUG EditFile: Nuevo contenido leído desde '%s', tamaño: %d bytes\n", params.Contenido, newSize)
	previewLen := 100
	if len(newContentStr) < previewLen {
		previewLen = len(newContentStr)
	}
	fmt.Printf("DEBUG EditFile: Primeros 100 caracteres del nuevo contenido: '%s'\n", newContentStr[:previewLen])

	// Liberar TODOS los bloques actuales del archivo (como removeOperations.go)
	fmt.Printf("DEBUG EditFile: Liberando TODOS los bloques del archivo (inodo %d, tamaño anterior: %d bytes)\n", inodeNum, inode.I_size)
	fmt.Printf("DEBUG EditFile: Bloques a liberar: %v\n", inode.I_block)
	err = freeInodeBlocksComplete(file, &inode, superblock)
	if err != nil {
		return fmt.Errorf("error al liberar bloques del archivo: %v", err)
	}
	fmt.Printf("DEBUG EditFile: Bloques liberados exitosamente\n")

	// Reiniciar COMPLETAMENTE el inodo
	inode.I_size = 0
	for i := 0; i < 15; i++ {
		inode.I_block[i] = 0
	}

	// Crear el contenido nuevo desde cero (como createFile.go)
	if newSize > 0 {
		fmt.Printf("DEBUG EditFile: Creando contenido nuevo (%d bytes)...\n", newSize)

		// Encontrar bloques libres para el contenido (usando búsqueda segura)
		blocksNeeded := (newSize + structs.FILE_CONTENT_SIZE - 1) / structs.FILE_CONTENT_SIZE
		if blocksNeeded == 0 {
			blocksNeeded = 1 // Al menos un bloque para archivos vacíos
		}

		freeBlocks, err := findSafeFreeBlocks(file, bmBlockPos, blockCount, blocksNeeded)
		if err != nil {
			return fmt.Errorf("error al encontrar bloques libres: %v", err)
		}

		// Asignar bloques al inodo manualmente para evitar problemas
		err = assignBlocksToInodeSafe(file, &inode, freeBlocks, blockPos, bmBlockPos)
		if err != nil {
			return fmt.Errorf("error al asignar bloques al inodo: %v", err)
		}

		fmt.Printf("DEBUG EditFile: Contenido nuevo creado exitosamente, bloques asignados: %v\n", inode.I_block[:5])
	}

	// Actualizar el tamaño y tiempo de modificación del inodo
	inode.I_size = newSize
	now := time.Now()
	inode.I_mtime = FormatTimeToBytes(now)

	// Escribir el inodo actualizado usando S_inode_start del superbloque
	inodeOffset = superblock.S_inode_start + (inodeNum * structs.INODE_SIZE)
	fmt.Printf("DEBUG EditFile: Escribiendo inodo %d actualizado en offset %d\n", inodeNum, inodeOffset)
	var inodeBuffer bytes.Buffer
	binary.Write(&inodeBuffer, binary.LittleEndian, inode)
	_, err = file.WriteAt(inodeBuffer.Bytes(), int64(inodeOffset))
	if err != nil {
		return fmt.Errorf("error al escribir el inodo actualizado: %v", err)
	}
	fmt.Printf("DEBUG EditFile: Inodo %d actualizado exitosamente\n", inodeNum)

	// Escribir el nuevo contenido en los bloques
	fmt.Printf("DEBUG EditFile: Escribiendo nuevo contenido (%d bytes) en los bloques del inodo\n", newSize)
	err = WriteContentToInodeBlocks(file, newContentStr, &inode, blockPos)
	if err != nil {
		return fmt.Errorf("error al escribir el nuevo contenido: %v", err)
	}
	fmt.Printf("DEBUG EditFile: Contenido escrito exitosamente en los bloques\n")

	// Registrar la operación en el journal EXT3
	journaling.LogEXT3Operation(session.ID, "EDIT_FILE", params.Path,
		fmt.Sprintf("Archivo editado - nuevo tamaño: %d bytes", len(newContentStr)))

	return nil
}

// hasReadAndWritePermission verifica si el usuario tiene permisos de lectura y escritura
func hasReadAndWritePermission(inode structs.Inode, user *structs.User) bool {
	if user == nil {
		return false
	}

	// Si es root, tiene todos los permisos
	if user.UID == 1 {
		return true
	}

	// Obtener los permisos del archivo
	perms := inode.I_perm

	// Verificar permisos de propietario
	if inode.I_uid == user.UID {
		hasRead := (perms & 0400) != 0
		hasWrite := (perms & 0200) != 0
		return hasRead && hasWrite
	}

	// Verificar permisos de grupo
	if inode.I_gid == user.GID {
		hasRead := (perms & 0040) != 0
		hasWrite := (perms & 0020) != 0
		return hasRead && hasWrite
	}

	// Verificar permisos de otros
	hasRead := (perms & 0004) != 0
	hasWrite := (perms & 0002) != 0
	return hasRead && hasWrite
}

// freeFileBlocks libera todos los bloques utilizados por un archivo
func freeFileBlocks(file *os.File, inode *structs.Inode, bmBlockPos, blockPos int32) error {
	// Liberar bloques directos (slots 0-11)
	for i := 0; i < 12; i++ {
		if inode.I_block[i] != 0 {
			err := markBlockAsFreeEdit(file, bmBlockPos, inode.I_block[i])
			if err != nil {
				return err
			}
		}
	}

	// Liberar bloque indirecto simple (slot 12)
	if inode.I_block[12] != 0 {
		err := freeIndirectBlockEdit(file, inode.I_block[12], bmBlockPos, blockPos)
		if err != nil {
			return err
		}
		err = markBlockAsFreeEdit(file, bmBlockPos, inode.I_block[12])
		if err != nil {
			return err
		}
	}

	// Liberar bloque doblemente indirecto (slot 13)
	if inode.I_block[13] != 0 {
		err := freeDoubleIndirectBlockEdit(file, inode.I_block[13], bmBlockPos, blockPos)
		if err != nil {
			return err
		}
		err = markBlockAsFreeEdit(file, bmBlockPos, inode.I_block[13])
		if err != nil {
			return err
		}
	}

	// Liberar bloque triplemente indirecto (slot 14)
	if inode.I_block[14] != 0 {
		err := freeTripleIndirectBlockEdit(file, inode.I_block[14], bmBlockPos, blockPos)
		if err != nil {
			return err
		}
		err = markBlockAsFreeEdit(file, bmBlockPos, inode.I_block[14])
		if err != nil {
			return err
		}
	}

	return nil
}

// markBlockAsFreeEdit marca un bloque como libre en el bitmap
func markBlockAsFreeEdit(file *os.File, bmBlockPos, blockNum int32) error {
	// Calcular el índice del byte y el bit dentro del bitmap
	byteIndex := blockNum / 8
	bitIndex := blockNum % 8

	// Leer el byte actual del bitmap
	currentByte := make([]byte, 1)
	_, err := file.ReadAt(currentByte, int64(bmBlockPos+byteIndex))
	if err != nil {
		return fmt.Errorf("error leyendo bitmap en byte %d: %v", byteIndex, err)
	}

	// Marcar el bit como libre (0)
	currentByte[0] &^= (1 << bitIndex) // Operación AND NOT para poner el bit en 0

	// Escribir el byte actualizado
	_, err = file.WriteAt(currentByte, int64(bmBlockPos+byteIndex))
	if err != nil {
		return fmt.Errorf("error escribiendo bitmap en byte %d: %v", byteIndex, err)
	}

	fmt.Printf("DEBUG markBlockAsFreeEdit: Bloque %d marcado como libre (byte=%d, bit=%d)\n", blockNum, byteIndex, bitIndex)
	return nil
}

// freeIndirectBlockEdit libera un bloque indirecto y todos sus bloques apuntados
func freeIndirectBlockEdit(file *os.File, blockNum int32, bmBlockPos, blockPos int32) error {
	// Leer el bloque de punteros
	ptrBlockBytes := make([]byte, structs.BLOCK_SIZE)
	_, err := file.ReadAt(ptrBlockBytes, int64(blockPos+(blockNum*structs.BLOCK_SIZE)))
	if err != nil {
		return err
	}

	var ptrBlock structs.PointerBlock
	buffer := bytes.NewReader(ptrBlockBytes)
	binary.Read(buffer, binary.LittleEndian, &ptrBlock)

	// Liberar cada bloque apuntado
	for _, ptr := range ptrBlock.B_pointers {
		if ptr != 0 {
			err = markBlockAsFreeEdit(file, bmBlockPos, ptr)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// freeDoubleIndirectBlockEdit libera un bloque doblemente indirecto
func freeDoubleIndirectBlockEdit(file *os.File, blockNum int32, bmBlockPos, blockPos int32) error {
	// Leer el bloque de punteros de primer nivel
	ptrBlockBytes := make([]byte, structs.BLOCK_SIZE)
	_, err := file.ReadAt(ptrBlockBytes, int64(blockPos+(blockNum*structs.BLOCK_SIZE)))
	if err != nil {
		return err
	}

	var ptrBlock structs.PointerBlock
	buffer := bytes.NewReader(ptrBlockBytes)
	binary.Read(buffer, binary.LittleEndian, &ptrBlock)

	// Liberar cada bloque indirecto apuntado
	for _, ptr := range ptrBlock.B_pointers {
		if ptr != 0 {
			err = freeIndirectBlockEdit(file, ptr, bmBlockPos, blockPos)
			if err != nil {
				return err
			}
			err = markBlockAsFreeEdit(file, bmBlockPos, ptr)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// freeTripleIndirectBlockEdit libera un bloque triplemente indirecto
func freeTripleIndirectBlockEdit(file *os.File, blockNum int32, bmBlockPos, blockPos int32) error {
	// Leer el bloque de punteros de primer nivel
	ptrBlockBytes := make([]byte, structs.BLOCK_SIZE)
	_, err := file.ReadAt(ptrBlockBytes, int64(blockPos+(blockNum*structs.BLOCK_SIZE)))
	if err != nil {
		return err
	}

	var ptrBlock structs.PointerBlock
	buffer := bytes.NewReader(ptrBlockBytes)
	binary.Read(buffer, binary.LittleEndian, &ptrBlock)

	// Liberar cada bloque doblemente indirecto apuntado
	for _, ptr := range ptrBlock.B_pointers {
		if ptr != 0 {
			err = freeDoubleIndirectBlockEdit(file, ptr, bmBlockPos, blockPos)
			if err != nil {
				return err
			}
			err = markBlockAsFreeEdit(file, bmBlockPos, ptr)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// freeInodeBlocksComplete libera TODOS los bloques de un inodo (basado en removeOperations.go)
func freeInodeBlocksComplete(file *os.File, inode *structs.Inode, sb *structs.Superblock) error {
	for i := 0; i < 12; i++ {
		if inode.I_block[i] != 0 {
			err := markBlockAsFreeRemove(file, inode.I_block[i], sb)
			if err != nil {
				return err
			}
		}
	}
	if inode.I_block[12] != 0 {
		err := freeIndirectBlockRemove(file, inode.I_block[12], sb)
		if err != nil {
			return err
		}
	}
	if inode.I_block[13] != 0 {
		err := freeDoubleIndirectBlockRemove(file, inode.I_block[13], sb)
		if err != nil {
			return err
		}
	}
	if inode.I_block[14] != 0 {
		err := freeTripleIndirectBlockRemove(file, inode.I_block[14], sb)
		if err != nil {
			return err
		}
	}
	return nil
}

func markBlockAsFreeRemove(file *os.File, blockIndex int32, sb *structs.Superblock) error {
	bmBlockPos := sb.S_bm_block_start
	byteIndex := blockIndex / 8
	bitIndex := blockIndex % 8
	currentByte := make([]byte, 1)
	_, err := file.ReadAt(currentByte, int64(bmBlockPos+byteIndex))
	if err != nil {
		return fmt.Errorf("error leyendo bitmap en byte %d: %v", byteIndex, err)
	}
	currentByte[0] &^= (1 << bitIndex)
	_, err = file.WriteAt(currentByte, int64(bmBlockPos+byteIndex))
	if err != nil {
		return fmt.Errorf("error escribiendo bitmap en byte %d: %v", byteIndex, err)
	}
	fmt.Printf("DEBUG markBlockAsFreeRemove: Bloque %d marcado como libre (byte=%d, bit=%d)\n", blockIndex, byteIndex, bitIndex)
	return nil
}

func freeIndirectBlockRemove(file *os.File, blockNum int32, sb *structs.Superblock) error {
	var blockPos int32
	if sb.S_block_start == 0 {
		blockPos = sb.S_inode_start + (sb.S_inodes_count * structs.INODE_SIZE) + (blockNum * structs.BLOCK_SIZE)
	} else {
		blockPos = sb.S_block_start + (blockNum * structs.BLOCK_SIZE)
	}
	blockData := make([]byte, structs.BLOCK_SIZE)
	_, err := file.ReadAt(blockData, int64(blockPos))
	if err != nil {
		return err
	}
	numPointers := int32(structs.BLOCK_SIZE / 4)
	for i := int32(0); i < numPointers; i++ {
		pointer := binary.LittleEndian.Uint32(blockData[i*4:])
		if pointer != 0 {
			err = markBlockAsFreeRemove(file, int32(pointer), sb)
			if err != nil {
				return err
			}
		}
	}
	return markBlockAsFreeRemove(file, blockNum, sb)
}

func freeDoubleIndirectBlockRemove(file *os.File, blockNum int32, sb *structs.Superblock) error {
	var blockPos int32
	if sb.S_block_start == 0 {
		blockPos = sb.S_inode_start + (sb.S_inodes_count * structs.INODE_SIZE) + (blockNum * structs.BLOCK_SIZE)
	} else {
		blockPos = sb.S_block_start + (blockNum * structs.BLOCK_SIZE)
	}
	blockData := make([]byte, structs.BLOCK_SIZE)
	_, err := file.ReadAt(blockData, int64(blockPos))
	if err != nil {
		return err
	}
	numPointers := int32(structs.BLOCK_SIZE / 4)
	for i := int32(0); i < numPointers; i++ {
		pointer := binary.LittleEndian.Uint32(blockData[i*4:])
		if pointer != 0 {
			err = freeIndirectBlockRemove(file, int32(pointer), sb)
			if err != nil {
				return err
			}
		}
	}
	return markBlockAsFreeRemove(file, blockNum, sb)
}

func freeTripleIndirectBlockRemove(file *os.File, blockNum int32, sb *structs.Superblock) error {
	var blockPos int32
	if sb.S_block_start == 0 {
		blockPos = sb.S_inode_start + (sb.S_inodes_count * structs.INODE_SIZE) + (blockNum * structs.BLOCK_SIZE)
	} else {
		blockPos = sb.S_block_start + (blockNum * structs.BLOCK_SIZE)
	}
	blockData := make([]byte, structs.BLOCK_SIZE)
	_, err := file.ReadAt(blockData, int64(blockPos))
	if err != nil {
		return err
	}
	numPointers := int32(structs.BLOCK_SIZE / 4)
	for i := int32(0); i < numPointers; i++ {
		pointer := binary.LittleEndian.Uint32(blockData[i*4:])
		if pointer != 0 {
			err = freeDoubleIndirectBlockRemove(file, int32(pointer), sb)
			if err != nil {
				return err
			}
		}
	}
	return markBlockAsFreeRemove(file, blockNum, sb)
}

// findSafeFreeBlocks busca bloques libres de manera segura para edición
func findSafeFreeBlocks(file *os.File, bmBlockPos, blockCount, blocksNeeded int32) ([]int32, error) {
	fmt.Printf("DEBUG findSafeFreeBlocks: Buscando %d bloques libres\n", blocksNeeded)

	var freeBlocks []int32

	// Empezar desde un bloque más alto para evitar conflictos con el sistema
	startBlock := int32(100) // Empezar desde el bloque 100 para mayor seguridad
	if startBlock >= blockCount {
		startBlock = blockCount / 2 // Si es muy pequeño, empezar desde la mitad
	}

	for i := startBlock; i < blockCount && int32(len(freeBlocks)) < blocksNeeded; i++ {
		// Leer el bitmap byte por byte
		currentByte := make([]byte, 1)
		_, err := file.ReadAt(currentByte, int64(bmBlockPos+i))
		if err != nil {
			return nil, fmt.Errorf("error leyendo bitmap en posición %d: %v", i, err)
		}

		// Si el byte es 0, el bloque está libre
		if currentByte[0] == 0 {
			freeBlocks = append(freeBlocks, i)
			fmt.Printf("DEBUG findSafeFreeBlocks: Bloque %d encontrado libre\n", i)
		}
	}

	if int32(len(freeBlocks)) < blocksNeeded {
		return nil, fmt.Errorf("no hay suficientes bloques libres (necesarios: %d, encontrados: %d)", blocksNeeded, len(freeBlocks))
	}

	fmt.Printf("DEBUG findSafeFreeBlocks: Encontrados bloques libres: %v\n", freeBlocks)
	return freeBlocks, nil
}

// assignBlocksToInodeSafe asigna bloques de manera segura sin usar funciones complejas
func assignBlocksToInodeSafe(file *os.File, inode *structs.Inode, blocks []int32, blockPos, bmBlockPos int32) error {
	fmt.Printf("DEBUG assignBlocksToInodeSafe: Asignando %d bloques: %v\n", len(blocks), blocks)

	// Asignar solo bloques directos (máximo 12)
	for i := 0; i < len(blocks) && i < 12; i++ {
		inode.I_block[i] = blocks[i]

		// Marcar el bloque como usado en el bitmap
		_, err := file.WriteAt([]byte{1}, int64(bmBlockPos+blocks[i]))
		if err != nil {
			return fmt.Errorf("error marcando bloque %d como usado: %v", blocks[i], err)
		}
		fmt.Printf("DEBUG assignBlocksToInodeSafe: Bloque %d marcado como usado\n", blocks[i])
	}

	if len(blocks) > 12 {
		return fmt.Errorf("archivo demasiado grande: se necesitan %d bloques, pero solo se soportan 12 en modo seguro", len(blocks))
	}

	return nil
}
