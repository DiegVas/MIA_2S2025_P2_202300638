package files

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"main/provider/common"
	"main/provider/mounted"
	"main/structs"
)

func CopyFileOrDirectory(params *structs.MoveParams) error {
	// Verificar que haya una sesión activa
	session := structs.GetSession()
	if !session.Active {
		return fmt.Errorf("debe iniciar sesión para usar el comando Copy")
	}

	// Obtener la partición montada
	mountManager := mounted.GetMountManager()
	partition, found := mountManager.GetMountedPartition(session.ID)
	if !found {
		return fmt.Errorf("no hay ninguna partición montada para esta sesión")
	}

	// Leer el MBR del disco
	mbr, err := mounted.ReadMBR(partition.Path)
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

	// Abrir el archivo del disco para lectura/escritura
	file, err := os.OpenFile(partition.Path, os.O_RDWR, 0666)
	if err != nil {
		return fmt.Errorf("error al abrir el archivo del disco: %v", err)
	}
	defer file.Close()

	// Verificar que el archivo o directorio origen existe
	sourceInodeNum, err := common.FindFileInodeByPath(file, params.Path, inodePos, blockPos)
	if err != nil {
		return fmt.Errorf("el archivo o directorio de origen no existe: %v", err)
	}

	// Leer el inodo del archivo/directorio origen para verificar permisos
	sourceInodeBytes := make([]byte, structs.INODE_SIZE)
	_, err = file.ReadAt(sourceInodeBytes, int64(inodePos+(sourceInodeNum*structs.INODE_SIZE)))
	if err != nil {
		return fmt.Errorf("error al leer el inodo del origen: %v", err)
	}

	var sourceInode structs.Inode
	buffer := bytes.NewReader(sourceInodeBytes)
	binary.Read(buffer, binary.LittleEndian, &sourceInode)

	// Verificar permisos de escritura sobre el archivo/directorio origen
	if !HasWritePermissionOnDirectory(file, sourceInodeNum, session, inodePos) {
		return errors.New("no tiene permisos de escritura sobre el archivo o directorio origen")
	}

	// Verificar que el directorio de destino existe
	destDir := filepath.Dir(params.Destino)
	destBaseName := filepath.Base(params.Destino)

	// Verificar que el directorio de destino existe
	destDirInodeNum, err := common.FindFileInodeByPath(file, destDir, inodePos, blockPos)
	if err != nil {
		return fmt.Errorf("el directorio de destino no existe: %v", err)
	}

	// Leer el inodo del directorio de destino
	destDirInodeBytes := make([]byte, structs.INODE_SIZE)
	_, err = file.ReadAt(destDirInodeBytes, int64(inodePos+(destDirInodeNum*structs.INODE_SIZE)))
	if err != nil {
		return fmt.Errorf("error al leer el inodo del directorio de destino: %v", err)
	}

	var destDirInode structs.Inode
	buffer = bytes.NewReader(destDirInodeBytes)
	binary.Read(buffer, binary.LittleEndian, &destDirInode)

	// Verificar que el destino es un directorio
	if destDirInode.I_type != 1 {
		return errors.New("el destino debe ser un directorio")
	}

	// Verificar permisos de escritura sobre el directorio de destino
	if !HasWritePermissionOnDirectory(file, destDirInodeNum, session, inodePos) {
		return errors.New("no tiene permisos de escritura sobre el directorio de destino")
	}

	// Verificar que no existe ya un archivo/directorio con el mismo nombre en el destino
	_, existsInDest := FindFileInDirectoryByInode(file, destBaseName, destDirInodeNum, inodePos, blockPos)
	if existsInDest {
		return fmt.Errorf("ya existe un archivo o directorio con el nombre '%s' en el destino", destBaseName)
	}

	// Realizar la copia del archivo o directorio:
	// 1. Agregar entrada en el directorio de destino
	err = AddFileToDirectoryWithIndirect(file, destBaseName, sourceInodeNum, destDirInodeNum, inodePos, blockPos, bmBlockPos, blockCount)
	if err != nil {
		return fmt.Errorf("error al agregar entrada en el directorio de destino: %v", err)
	}

	return nil
}
