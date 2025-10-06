package users

import (
	"bytes"
	"encoding/binary"
	"fmt"
	diskmanager "main/diskManager"
	"main/structs"
	"os"
	"strings"
	"unsafe"
)

func ReadUsersFile(diskPath, partitionName string) (string, error) {
	// ? Leer el MBR del disco
	mbr, err := diskmanager.ReadMBR(diskPath)
	if err != nil {
		return "", fmt.Errorf("error al leer el MBR del disco: %v", err)
	}

	// ? Buscar la partición
	var targetPartition structs.PARTITION
	found := false

	for _, p := range mbr.Partitions {
		partName := strings.TrimRight(string(p.Part_name[:]), "\x00")
		if partName == partitionName {
			targetPartition = p
			found = true
			break
		}
	}

	if !found {
		return "", fmt.Errorf("la partición '%s' no se encontró en el disco", partitionName)
	}

	// ? Abrir el archivo del disco
	file, err := os.Open(diskPath)
	if err != nil {
		return "", fmt.Errorf("error al abrir el disco: %v", err)
	}
	defer file.Close()

	// Detectar el tipo de sistema de archivos leyendo el superbloque
	startPosition := targetPartition.Part_start

	// Debug: verificar el tamaño del superbloque
	fmt.Printf("DEBUG: SUPERBLOCK_SIZE definido: %d\n", structs.SUPERBLOCK_SIZE)
	realSuperbSize := int(unsafe.Sizeof(structs.Superblock{}))
	fmt.Printf("DEBUG: Tamaño real del struct Superblock: %d\n", realSuperbSize)

	// Usar el tamaño real del struct en lugar de la constante
	superbBytes := make([]byte, realSuperbSize)
	_, err = file.ReadAt(superbBytes, int64(startPosition))
	if err != nil {
		return "", fmt.Errorf("error al leer el superbloque: %v", err)
	}

	fmt.Printf("DEBUG: Bytes realmente leídos: %d\n", len(superbBytes)) // Deserializar el superbloque para obtener el tipo de sistema de archivos
	var superblock structs.Superblock
	superbBuffer := bytes.NewReader(superbBytes)
	err = binary.Read(superbBuffer, binary.LittleEndian, &superblock)
	if err != nil {
		return "", fmt.Errorf("error al deserializar el superbloque: %v", err)
	}

	fmt.Printf("DEBUG LECTURA: Posición de lectura del superbloque: %d\n", startPosition)
	fmt.Printf("DEBUG LECTURA: Bytes leídos del superbloque: %v\n", superbBytes[:20]) // Primeros 20 bytes
	fmt.Printf("DEBUG LECTURA: Tipo de sistema de archivos detectado: %d\n", superblock.S_filesystem_type)
	fmt.Printf("DEBUG LECTURA: Inodes count: %d\n", superblock.S_inodes_count)
	fmt.Printf("DEBUG LECTURA: Blocks count: %d\n", superblock.S_blocks_count)
	fmt.Printf("DEBUG LECTURA: Magic number: %d\n", superblock.S_magic)
	fmt.Printf("DEBUG LECTURA: Partición start: %d, size: %d\n", targetPartition.Part_start, targetPartition.Part_size)

	// Debug manual de los primeros bytes
	if len(superbBytes) >= 4 {
		filesystemType := int32(superbBytes[0]) | int32(superbBytes[1])<<8 | int32(superbBytes[2])<<16 | int32(superbBytes[3])<<24
		fmt.Printf("DEBUG LECTURA: Tipo de filesystem manual: %d\n", filesystemType)
	}

	// Verificar si la partición está formateada
	if superblock.S_filesystem_type == 0 {
		return "", fmt.Errorf("la partición '%s' no está formateada. Use el comando 'mkfs' para formatearla primero", partitionName)
	}

	// Verificar que sea un sistema de archivos válido
	if superblock.S_filesystem_type != 2 && superblock.S_filesystem_type != 3 {
		return "", fmt.Errorf("tipo de sistema de archivos no válido: %d. Solo se soportan EXT2 (2) y EXT3 (3)", superblock.S_filesystem_type)
	}

	var inodePos, blockPos int32

	// Usar las posiciones almacenadas en el superbloque para ambos tipos
	inodePos = superblock.S_inode_start
	blockPos = superblock.S_block_start

	fmt.Printf("DEBUG ReadUsersFile: Usando posiciones del superbloque - inodePos=%d, blockPos=%d\n", inodePos, blockPos)

	// Posición del inodo de users.txt (inodo 1)
	usersInodePos := inodePos + structs.INODE_SIZE

	fmt.Printf("DEBUG: usersInodePos=%d\n", usersInodePos)

	// ? Leer el inodo de users.txt
	inodeBytes := make([]byte, structs.INODE_SIZE)
	_, err = file.ReadAt(inodeBytes, int64(usersInodePos))
	if err != nil {
		return "", fmt.Errorf("error al leer el inodo de users.txt: %v", err)
	}

	// Deserializar el inodo
	var inode structs.Inode
	buffer := bytes.NewReader(inodeBytes)
	binary.Read(buffer, binary.LittleEndian, &inode)

	fmt.Printf("DEBUG: Inodo leído - Size: %d, Type: %d, Block[0]: %d\n", inode.I_size, inode.I_type, inode.I_block[0])

	// Leer el contenido usando múltiples bloques
	content, err := readUsersFileData(file, inode, blockPos)
	if err != nil {
		return "", fmt.Errorf("error al leer el contenido: %v", err)
	}

	fmt.Printf("DEBUG: Contenido leído: '%s'\n", content)

	return content, nil
}

// readUsersFileData lee el contenido del archivo users.txt usando múltiples bloques
func readUsersFileData(file *os.File, inode structs.Inode, blockPos int32) (string, error) {
	var content strings.Builder
	totalBytesRead := 0

	fmt.Printf("DEBUG: readUsersFileData - blockPos: %d, inode.I_size: %d\n", blockPos, inode.I_size)

	// Leer bloques directos (slots 0-11)
	for i := 0; i < 12 && inode.I_block[i] != 0 && totalBytesRead < int(inode.I_size); i++ {
		blockNum := inode.I_block[i]

		fmt.Printf("DEBUG: Leyendo bloque %d en posición %d\n", blockNum, blockPos+(blockNum*structs.BLOCK_SIZE))

		// Leer el bloque completo
		blockBytes := make([]byte, structs.BLOCK_SIZE)
		_, err := file.ReadAt(blockBytes, int64(blockPos+(blockNum*structs.BLOCK_SIZE)))
		if err != nil {
			return "", fmt.Errorf("error al leer el bloque %d: %v", blockNum, err)
		}

		// Determinar cuántos bytes leer de este bloque
		bytesToRead := structs.FILE_CONTENT_SIZE
		if totalBytesRead+bytesToRead > int(inode.I_size) {
			bytesToRead = int(inode.I_size) - totalBytesRead
		}

		fmt.Printf("DEBUG: Bytes a leer del bloque: %d, contenido: '%s'\n", bytesToRead, string(blockBytes[:bytesToRead]))

		// Agregar al contenido
		content.Write(blockBytes[:bytesToRead])
		totalBytesRead += bytesToRead
	}

	fmt.Printf("DEBUG: Contenido final leído: '%s', bytes totales: %d\n", content.String(), totalBytesRead)

	return content.String(), nil
}
