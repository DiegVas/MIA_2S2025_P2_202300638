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

	diskmanager "main/diskManager"
	"main/provider/common"
	"main/provider/mounted"
	"main/provider/users"
	"main/structs"
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

	// Leer el MBR del disco
	mbr, err := diskmanager.ReadMBR(partition.Path)
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

	// Calcular las posiciones de las estructuras del sistema de archivos
	startPosition := targetPartition.Part_start
	inodeCount := int32(float64(targetPartition.Part_size) * 0.05 / float64(structs.INODE_SIZE))
	blockCount := (targetPartition.Part_size - inodeCount*structs.INODE_SIZE) / structs.BLOCK_SIZE

	bmInodePos := startPosition + structs.SUPERBLOCK_SIZE
	bmBlockPos := bmInodePos + inodeCount
	inodePos := bmBlockPos + blockCount
	blockPos := inodePos + (inodeCount * structs.INODE_SIZE)

	// Abrir el archivo del disco para escritura
	file, err := os.OpenFile(partition.Path, os.O_RDWR, 0666)
	if err != nil {
		return fmt.Errorf("error al abrir el disco: %v", err)
	}
	defer file.Close()

	// Buscar el inodo del archivo a editar
	inodeNum, err := common.FindFileInodeByPath(file, params.Path, inodePos, blockPos)
	if err != nil {
		return fmt.Errorf("archivo no encontrado: %v", err)
	}

	// Leer el inodo del archivo
	inodeBytes := make([]byte, structs.INODE_SIZE)
	_, err = file.ReadAt(inodeBytes, int64(inodePos+(inodeNum*structs.INODE_SIZE)))
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

	// Liberar los bloques actuales del archivo
	err = freeFileBlocks(file, &inode, bmBlockPos, blockPos)
	if err != nil {
		return fmt.Errorf("error al liberar bloques del archivo: %v", err)
	}

	// Asignar nuevos bloques para el nuevo contenido
	blocksNeeded := (newSize + structs.FILE_CONTENT_SIZE - 1) / structs.FILE_CONTENT_SIZE
	if blocksNeeded == 0 {
		blocksNeeded = 1 // Al menos un bloque para archivos vacíos
	}

	freeBlocks, err := FindFreeBlocks(file, bmBlockPos, blockCount, blocksNeeded)
	if err != nil {
		return fmt.Errorf("error al buscar bloques libres: %v", err)
	}

	// Reiniciar los bloques del inodo
	for i := 0; i < 15; i++ {
		inode.I_block[i] = 0
	}

	// Asignar los nuevos bloques al inodo
	err = assignBlocksToInode(file, &inode, freeBlocks, blockPos, bmBlockPos, blockCount)
	if err != nil {
		return fmt.Errorf("error al asignar bloques al inodo: %v", err)
	}

	// Actualizar el tamaño y tiempo de modificación del inodo
	inode.I_size = newSize
	now := time.Now()
	inode.I_mtime = FormatTimeToBytes(now)

	// Escribir el inodo actualizado
	var inodeBuffer bytes.Buffer
	binary.Write(&inodeBuffer, binary.LittleEndian, inode)
	_, err = file.WriteAt(inodeBuffer.Bytes(), int64(inodePos+(inodeNum*structs.INODE_SIZE)))
	if err != nil {
		return fmt.Errorf("error al escribir el inodo actualizado: %v", err)
	}

	// Escribir el nuevo contenido en los bloques
	err = WriteContentToInodeBlocks(file, newContentStr, &inode, blockPos)
	if err != nil {
		return fmt.Errorf("error al escribir el nuevo contenido: %v", err)
	}

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
	position := int64(bmBlockPos + blockNum)
	_, err := file.WriteAt([]byte{0}, position)
	return err
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
