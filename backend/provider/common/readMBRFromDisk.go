package common

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"main/structs"
	"os"
)

// readMBRFromDisk lee el MBR desde el archivo del disco usando el mismo método que disk_partition.go
func ReadMBRFromDisk(diskPath string) (*structs.MBR, error) {
	file, err := os.Open(diskPath)
	if err != nil {
		return nil, fmt.Errorf("error abriendo archivo del disco: %v", err)
	}
	defer file.Close()

	// Usar la misma lógica que disk_partition.go
	const MBR_SIZE = 47

	mbrBytes := make([]byte, MBR_SIZE)
	_, err = file.ReadAt(mbrBytes, 0)
	if err != nil {
		return nil, fmt.Errorf("error al leer el MBR: %v", err)
	}

	var mbr structs.MBR
	buffer := bytes.NewReader(mbrBytes)

	// Leer cabecera del MBR
	binary.Read(buffer, binary.LittleEndian, &mbr.Mbr_size)
	buffer.Read(mbr.Mbr_timestamp[:])
	buffer.Read(mbr.Mbr_uuid[:])
	binary.Read(buffer, binary.LittleEndian, &mbr.Mbr_partitions)
	buffer.Read(mbr.Dsk_fit[:])

	// Leer particiones por separado
	partitionDataOffset := int64(MBR_SIZE)
	partitionSize := int64(59) // Tamaño de cada estructura PARTITION

	for i := 0; i < 4; i++ {
		partPos := partitionDataOffset + int64(i)*partitionSize

		partData := make([]byte, partitionSize)
		_, err = file.ReadAt(partData, partPos)
		if err != nil {
			return nil, fmt.Errorf("error al leer la partición %d: %v", i, err)
		}

		partBuffer := bytes.NewReader(partData)

		partBuffer.Read(mbr.Partitions[i].Part_status[:])
		partBuffer.Read(mbr.Partitions[i].Part_type[:])
		partBuffer.Read(mbr.Partitions[i].Part_fit[:])
		binary.Read(partBuffer, binary.LittleEndian, &mbr.Partitions[i].Part_start)
		binary.Read(partBuffer, binary.LittleEndian, &mbr.Partitions[i].Part_size)
		partBuffer.Read(mbr.Partitions[i].Part_name[:])
		partBuffer.Read(mbr.Partitions[i].Part_uuid[:])
		binary.Read(partBuffer, binary.LittleEndian, &mbr.Partitions[i].Part_correlative)
	}

	return &mbr, nil
}
