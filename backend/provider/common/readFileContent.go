package common

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"

	diskmanager "main/diskManager"
	"main/provider/mounted"
	"main/provider/users"
	"main/structs"
	"os"
	"strings"
	"unsafe"
)

func ReadFileContent(filePath string, idPartition string) (string, error) {
	// Verificar que haya una sesión activa
	session := structs.GetSession()
	if !session.Active && idPartition == "" {
		return "", errors.New("debe iniciar sesión para usar el comando CAT")
	} // Obtener la partición montada

	idP := ""

	if idPartition != "" {
		idP = idPartition
	} else {
		idP = session.ID
	}

	mountManager := mounted.GetMountManager()
	partition, partitionFound := mountManager.GetMountedPartition(idP)
	if !partitionFound {
		return "", errors.New("no hay ninguna partición montada para esta sesión")
	}

	// Leer el MBR del disco
	mbr, err := diskmanager.ReadMBR(partition.Path)
	if err != nil {
		return "", fmt.Errorf("error al leer el MBR del disco: %v", err)
	}

	// Buscar la partición montada
	var targetPartition structs.PARTITION
	var found bool
	for _, p := range mbr.Partitions {
		partName := strings.TrimRight(string(p.Part_name[:]), "\x00")
		if partName == partition.Name {
			targetPartition = p
			found = true
			break
		}
	}
	if !found {
		return "", fmt.Errorf("la partición '%s' no se encontró en el disco", partition.Name)
	}

	// Abrir el archivo del disco
	file, err := os.Open(partition.Path)
	if err != nil {
		return "", fmt.Errorf("error al abrir el disco: %v", err)
	}
	defer file.Close()

	// Detectar el tipo de sistema de archivos leyendo el superbloque
	startPosition := targetPartition.Part_start
	realSuperbSize := int(unsafe.Sizeof(structs.Superblock{}))
	superbBytes := make([]byte, realSuperbSize)
	_, err = file.ReadAt(superbBytes, int64(startPosition))
	if err != nil {
		return "", fmt.Errorf("error al leer el superbloque: %v", err)
	}

	// Deserializar el superbloque para obtener el tipo de sistema de archivos
	var superblock structs.Superblock
	superbBuffer := bytes.NewReader(superbBytes)
	err = binary.Read(superbBuffer, binary.LittleEndian, &superblock)
	if err != nil {
		return "", fmt.Errorf("error al deserializar el superbloque: %v", err)
	}

	// Verificar si la partición está formateada
	if superblock.S_filesystem_type == 0 {
		return "", fmt.Errorf("la partición '%s' no está formateada. Use el comando 'mkfs' para formatearla primero", partition.Name)
	}

	// Verificar que sea un sistema de archivos válido
	if superblock.S_filesystem_type != 2 && superblock.S_filesystem_type != 3 {
		return "", fmt.Errorf("tipo de sistema de archivos no válido: %d. Solo se soportan EXT2 (2) y EXT3 (3)", superblock.S_filesystem_type)
	}

	var inodePos, blockPos int32

	if superblock.S_filesystem_type == 3 {
		// EXT3: Usar las posiciones del superbloque
		inodePos = superblock.S_inode_start
		blockPos = superblock.S_block_start
	} else {
		// EXT2: Usar las posiciones del superbloque
		inodePos = superblock.S_inode_start
		blockPos = superblock.S_block_start
	}

	fmt.Printf("DEBUG ReadFileContent: filesystem_type=%d, inodePos=%d, blockPos=%d\n", superblock.S_filesystem_type, inodePos, blockPos)
	fmt.Printf("DEBUG ReadFileContent SUPERBLOQUE: S_inode_start=%d, S_block_start=%d\n",
		superblock.S_inode_start, superblock.S_block_start)

	// Navegar por el path y encontrar el archivo
	inodeNum, err := FindFileInodeByPath(file, filePath, &superblock, inodePos, blockPos)
	if err != nil {
		return "", err
	}

	// Leer el inodo del archivo
	inodeBytes := make([]byte, structs.INODE_SIZE)
	_, err = file.ReadAt(inodeBytes, int64(inodePos+(inodeNum*structs.INODE_SIZE)))
	if err != nil {
		return "", fmt.Errorf("error al leer el inodo del archivo: %v", err)
	}

	// Deserializar el inodo
	var inode structs.Inode
	buffer := bytes.NewReader(inodeBytes)
	binary.Read(buffer, binary.LittleEndian, &inode)

	// Verificar que es un archivo (no un directorio)
	if inode.I_type != 0 {
		return "", errors.New("el path especificado no es un archivo")
	}

	// Verificar permisos de lectura
	currentUser, err := users.GetCurrentUserInfo(session, partition)
	if err != nil && idPartition == "" {
		return "", fmt.Errorf("error al obtener información del usuario: %v", err)
	}

	if !HasReadPermission(inode, currentUser) && idPartition == "" {
		return "", errors.New("no tiene permisos de lectura para este archivo")
	}

	// Leer el contenido del archivo
	content, err := ReadFileData(file, inode, blockPos)
	if err != nil {
		return "", fmt.Errorf("error al leer el contenido del archivo: %v", err)
	}

	return content, nil
}
