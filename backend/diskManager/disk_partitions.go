package diskmanager

import (
	"fmt"
	"main/provider/mounted"
	"main/structs"
	"os"
	"strings"
)

// GetDiskPartitions obtiene información de todas las particiones de un disco
func GetDiskPartitions(diskPath string) (*structs.DiskPartitionsInfo, error) {
	// Verificar que el disco existe
	if _, err := os.Stat(diskPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("el disco no existe: %s", diskPath)
	}

	// Leer el MBR del disco para obtener información de particiones
	mbr, err := ReadMBR(diskPath)
	if err != nil {
		return nil, fmt.Errorf("error al leer MBR: %v", err)
	}

	// Crear la estructura de respuesta
	diskInfo := &structs.DiskPartitionsInfo{
		DiskPath:   diskPath,
		Partitions: []structs.PartitionInfo{},
	}

	// Obtener lista de particiones montadas para verificar el estado
	mountManager := mounted.GetMountManager()
	mountedPartitions := mountManager.GetMountedPartitions()

	// Procesar particiones primarias y extendidas
	for i := 0; i < 4; i++ {
		partition := mbr.Partitions[i]

		// Verificar si la partición está en uso (Part_status[0] != 0)
		if partition.Part_status[0] != 0 {
			// Determinar el tipo de partición
			partType := "Primaria"
			if partition.Part_type[0] == 'E' {
				partType = "Extendida"
			}

			// Limpiar el nombre de la partición (quitar bytes nulos)
			partitionName := strings.TrimRight(string(partition.Part_name[:]), "\x00")

			// Verificar si está montada
			status := "Desmontada"
			for _, mounted := range mountedPartitions {
				if mounted.Path == diskPath && mounted.Name == partitionName {
					status = "Montada"
					break
				}
			}

			// Crear información de la partición
			partInfo := structs.PartitionInfo{
				Name:   partitionName,
				Type:   partType,
				Status: status,
				Fit:    strings.TrimRight(string(partition.Part_fit[:]), "\x00"),
				Start:  partition.Part_start,
				Size:   partition.Part_size,
			}

			diskInfo.Partitions = append(diskInfo.Partitions, partInfo)

			// Si es extendida, procesar particiones lógicas
			if partition.Part_type[0] == 'E' {
				logicalPartitions, err := getLogicalPartitions(diskPath, partition.Part_start, mountedPartitions)
				if err == nil {
					diskInfo.Partitions = append(diskInfo.Partitions, logicalPartitions...)
				}
			}
		}
	}

	return diskInfo, nil
}

// getLogicalPartitions obtiene las particiones lógicas de una partición extendida
func getLogicalPartitions(diskPath string, extendedStart int32, mountedPartitions []mounted.MountedPartition) ([]structs.PartitionInfo, error) {
	var logicalPartitions []structs.PartitionInfo
	currentEBRPos := extendedStart

	// Abrir el disco para leer los EBR
	diskFile, err := os.Open(diskPath)
	if err != nil {
		return nil, fmt.Errorf("error al abrir disco para leer EBR: %v", err)
	}
	defer diskFile.Close()

	for currentEBRPos != 0 {
		var ebr structs.EBR
		err := ReadEBR(diskFile, &ebr, int64(currentEBRPos))
		if err != nil {
			break
		}

		// Verificar si el EBR contiene una partición válida
		if ebr.Part_mount[0] != 0 {
			// Limpiar el nombre de la partición
			partitionName := strings.TrimRight(string(ebr.Part_name[:]), "\x00")

			// Verificar si está montada
			status := "Desmontada"
			for _, mounted := range mountedPartitions {
				if mounted.Path == diskPath && mounted.Name == partitionName {
					status = "Montada"
					break
				}
			}

			// Crear información de la partición lógica
			partInfo := structs.PartitionInfo{
				Name:   partitionName,
				Type:   "Lógica",
				Status: status,
				Fit:    strings.TrimRight(string(ebr.Part_fit[:]), "\x00"),
				Start:  ebr.Part_start,
				Size:   ebr.Part_size,
			}

			logicalPartitions = append(logicalPartitions, partInfo)
		}

		// Ir al siguiente EBR
		currentEBRPos = ebr.Part_next
	}

	return logicalPartitions, nil
}

// ReadEBR lee un Extended Boot Record desde un archivo de disco
func ReadEBR(diskFile *os.File, ebr *structs.EBR, position int64) error {
	ebrSize := int32(30) // Tamaño del EBR según la estructura
	ebrData := make([]byte, ebrSize)

	_, err := diskFile.ReadAt(ebrData, position)
	if err != nil {
		return fmt.Errorf("error leyendo EBR en posición %d: %v", position, err)
	}

	// Parsear los campos del EBR
	ebr.Part_mount[0] = ebrData[0]
	ebr.Part_fit[0] = ebrData[1]

	// Leer Part_start (4 bytes, little endian)
	ebr.Part_start = int32(ebrData[2]) | int32(ebrData[3])<<8 | int32(ebrData[4])<<16 | int32(ebrData[5])<<24

	// Leer Part_size (4 bytes, little endian)
	ebr.Part_size = int32(ebrData[6]) | int32(ebrData[7])<<8 | int32(ebrData[8])<<16 | int32(ebrData[9])<<24

	// Leer Part_next (4 bytes, little endian)
	ebr.Part_next = int32(ebrData[10]) | int32(ebrData[11])<<8 | int32(ebrData[12])<<16 | int32(ebrData[13])<<24

	// Copiar nombre (16 bytes)
	copy(ebr.Part_name[:], ebrData[14:30])

	return nil
}
