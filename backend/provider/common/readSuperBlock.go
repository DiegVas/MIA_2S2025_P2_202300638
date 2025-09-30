package common

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"main/provider/mounted"
	"main/structs"
	"os"
	"strings"
)

func ReadSuperblock(diskPath, partitionID string) (*structs.Superblock, error) {

	// Obtener información de la partición montada para encontrar el superbloque
	mountManager := mounted.GetMountManager()
	mountedPartition, found := mountManager.GetMountedPartition(partitionID)
	if !found {
		return nil, fmt.Errorf("partición no encontrada")
	}

	// Leer el MBR para encontrar la partición específica
	mbr, err := ReadMBRFromDisk(diskPath)
	if err != nil {
		return nil, fmt.Errorf("error leyendo MBR: %v", err)
	}

	// Buscar la partición por nombre
	var targetPartition structs.PARTITION
	found = false

	for i, partition := range mbr.Partitions {
		fmt.Printf("Partición %d: %s\n", i, partition.Part_name)
		partName := strings.TrimRight(string(partition.Part_name[:]), "\x00")
		if partName == mountedPartition.Name {
			targetPartition = partition
			found = true
			break
		}
	}

	if !found {
		return nil, fmt.Errorf("partición '%s' no encontrada en el disco", mountedPartition.Name)
	}

	// Verificar que la particion tenga un tamano valido
	if targetPartition.Part_size <= 0 {
		return nil, fmt.Errorf("particion '%s' tiene un tamano invalido: %d", mountedPartition.Name, targetPartition.Part_size)
	}

	// Abrir el archivo del disco
	file, err := os.Open(diskPath)
	if err != nil {
		return nil, fmt.Errorf("error abriendo disco: %v", err)
	}
	defer file.Close()

	// Verificar el tamano del archivo
	fileInfo, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("error obteniendo información del archivo: %v", err)
	}

	// Verificar que la posición de lectura esté dentro del archivo
	readPos := int64(targetPartition.Part_start)

	if readPos+int64(structs.SUPERBLOCK_SIZE) > fileInfo.Size() {
		return nil, fmt.Errorf("posición de lectura fuera del archivo: pos=%d, size=%d, superblock_size=%d",
			readPos, fileInfo.Size(), structs.SUPERBLOCK_SIZE)
	}

	// Leer el superbloque desde la posición de inicio de la partición
	superbData := make([]byte, structs.SUPERBLOCK_SIZE)
	bytesRead, err := file.ReadAt(superbData, readPos)
	if err != nil {
		return nil, fmt.Errorf("error leyendo superbloque: %v", err)
	}

	if bytesRead != structs.SUPERBLOCK_SIZE {
		return nil, fmt.Errorf("no se pudieron leer todos los bytes del superbloque: leídos=%d, esperados=%d",
			bytesRead, structs.SUPERBLOCK_SIZE)
	}

	// Deserializar el superbloque campo por campo para evitar problemas de alineación
	var superblock structs.Superblock
	buffer := bytes.NewReader(superbData)

	// Leer campos uno por uno
	binary.Read(buffer, binary.LittleEndian, &superblock.S_filesystem_type)
	binary.Read(buffer, binary.LittleEndian, &superblock.S_inodes_count)
	binary.Read(buffer, binary.LittleEndian, &superblock.S_blocks_count)
	binary.Read(buffer, binary.LittleEndian, &superblock.S_free_blocks_count)
	binary.Read(buffer, binary.LittleEndian, &superblock.S_free_inodes_count)
	buffer.Read(superblock.S_mtime[:])
	buffer.Read(superblock.S_umtime[:])
	binary.Read(buffer, binary.LittleEndian, &superblock.S_mnt_count)
	binary.Read(buffer, binary.LittleEndian, &superblock.S_magic)
	binary.Read(buffer, binary.LittleEndian, &superblock.S_inode_size)
	binary.Read(buffer, binary.LittleEndian, &superblock.S_block_size)
	binary.Read(buffer, binary.LittleEndian, &superblock.S_first_ino)
	binary.Read(buffer, binary.LittleEndian, &superblock.S_first_blo)
	binary.Read(buffer, binary.LittleEndian, &superblock.S_bm_inode_start)
	binary.Read(buffer, binary.LittleEndian, &superblock.S_bm_block_start)
	binary.Read(buffer, binary.LittleEndian, &superblock.S_inode_start)
	binary.Read(buffer, binary.LittleEndian, &superblock.S_block_start)

	// Verificar que el superbloque tenga valores válidos
	if superblock.S_magic != structs.EXT2_MAGIC {
		return nil, fmt.Errorf("la partición '%s' no tiene un sistema de archivos EXT2 válido. Número mágico: 0x%X, esperado: 0x%X. Use 'mkfs' para formatear la partición",
			mountedPartition.Name, superblock.S_magic, structs.EXT2_MAGIC)
	}

	if superblock.S_inodes_count <= 0 {
		return nil, fmt.Errorf("número de inodos inválido: %d", superblock.S_inodes_count)
	}

	return &superblock, nil
}
