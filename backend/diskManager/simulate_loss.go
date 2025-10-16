package diskmanager

import (
	"fmt"
	"main/provider/mounted"
	"os"
)

// SimulateLoss limpia bloques específicos para simular la pérdida del sistema de archivos EXT3.
func SimulateLoss(diskPath string, partitionStart int64, partitionSize int64) error {
	fmt.Printf("Simulando pérdida en la partición: %s\n", diskPath)

	// Abrir el archivo del disco
	diskFile, err := os.OpenFile(diskPath, os.O_RDWR, 0644)
	if err != nil {
		return fmt.Errorf("error al abrir el disco: %v", err)
	}
	defer diskFile.Close()

	// Limpiar los bloques especificados
	zeroBlock := make([]byte, 1024) // Bloque de ceros de 1 KB
	for offset := partitionStart; offset < partitionStart+partitionSize; offset += 1024 {
		_, err := diskFile.WriteAt(zeroBlock, offset)
		if err != nil {
			return fmt.Errorf("error al limpiar bloque en offset %d: %v", offset, err)
		}
	}

	fmt.Println("Pérdida simulada exitosamente.")
	return nil
}

// SimulateFileSystemLoss simula la pérdida del sistema de archivos EXT3 en una partición
func SimulateFileSystemLoss(diskPath string, partitionName string) error {
	fmt.Printf("Simulando pérdida del sistema de archivos EXT3 en partición %s del disco %s\n", partitionName, diskPath)

	// Obtener información de la partición montada
	mountManager := mounted.GetMountManager()
	partitions := mountManager.GetMountedPartitions()

	var targetPartition *mounted.MountedPartition
	for _, partition := range partitions {
		if partition.Path == diskPath && partition.Name == partitionName {
			targetPartition = &partition
			break
		}
	}

	if targetPartition == nil {
		return fmt.Errorf("no se encontró la partición %s en el disco %s", partitionName, diskPath)
	}

	// Abrir el archivo del disco para escritura
	diskFile, err := os.OpenFile(diskPath, os.O_RDWR, 0644)
	if err != nil {
		return fmt.Errorf("error al abrir el disco para escritura: %v", err)
	}
	defer diskFile.Close()

	// Necesitamos encontrar la partición en el disco y leer su superbloque
	// Para esto, primero leemos el MBR para encontrar la partición
	err = clearEXT3SystemBlocks(diskFile, diskPath, partitionName)
	if err != nil {
		return fmt.Errorf("error al limpiar bloques del sistema EXT3: %v", err)
	}

	// Agregar entrada al journal para registrar la pérdida
	AddJournalEntry("SYSTEM_LOSS", fmt.Sprintf("Partición %s", partitionName), "Simulación de pérdida del sistema de archivos EXT3 - Bloques críticos limpiados")

	fmt.Printf("Pérdida del sistema de archivos simulada para partición %s - Bloques críticos limpiados\n", partitionName)
	return nil
}

// clearEXT3SystemBlocks limpia los bloques críticos del sistema de archivos EXT3
func clearEXT3SystemBlocks(diskFile *os.File, diskPath string, partitionName string) error {
	fmt.Println("Iniciando limpieza de bloques críticos del sistema EXT3...")

	// Usar la función existente ReadMBR del proyecto
	mbr, err := ReadMBR(diskPath)
	if err != nil {
		return fmt.Errorf("error al leer MBR: %v", err)
	}

	// Buscar la partición por nombre
	var partitionStart int64 = -1
	var partitionSize int64 = -1

	// Buscar en particiones primarias
	for i := 0; i < 4; i++ {
		partition := mbr.Partitions[i]
		if partition.Part_status[0] != 0 {
			// Limpiar el nombre de la partición
			name := string(partition.Part_name[:])
			cleanName := ""
			for j := 0; j < len(name) && name[j] != 0; j++ {
				cleanName += string(name[j])
			}

			fmt.Printf("DEBUG: Comparando partición '%s' con objetivo '%s'\n", cleanName, partitionName)

			if cleanName == partitionName {
				partitionStart = int64(partition.Part_start)
				partitionSize = int64(partition.Part_size)
				fmt.Printf("DEBUG: Partición encontrada - Start: %d, Size: %d\n", partitionStart, partitionSize)
				break
			}
		}
	}

	if partitionStart == -1 {
		return fmt.Errorf("no se encontró la partición %s en el disco", partitionName)
	}

	fmt.Printf("Partición encontrada: inicio=%d, tamaño=%d\n", partitionStart, partitionSize)

	// LIMPIEZA AGRESIVA TOTAL DEL SISTEMA DE ARCHIVOS EXT3
	blockSize := int64(1024) // Tamaño estándar de bloque EXT3

	// 1. Limpiar COMPLETAMENTE desde el inicio de la partición
	fmt.Printf("DEBUG: DESTRUCCIÓN TOTAL - Limpiando desde el inicio de la partición\n")
	err = clearArea(diskFile, partitionStart, blockSize*8) // Limpiar primeros 8KB
	if err != nil {
		fmt.Printf("ADVERTENCIA: No se pudo limpiar inicio de partición: %v\n", err)
	} else {
		fmt.Println("✓ Inicio de partición DESTRUIDO")
	}

	// 2. Limpiar MASIVAMENTE el superbloque y área extendida
	superblockOffset := partitionStart + 1024
	fmt.Printf("DEBUG: DESTRUCCIÓN TOTAL - Limpiando superbloque extendido desde posición %d\n", superblockOffset)
	err = clearArea(diskFile, superblockOffset, blockSize*10) // Limpiar 10KB alrededor del superbloque
	if err != nil {
		fmt.Printf("ADVERTENCIA: No se pudo destruir superbloque: %v\n", err)
	} else {
		fmt.Println("✓ Superbloque COMPLETAMENTE DESTRUIDO")
	}

	// 3. Limpiar TODA el área de metadatos críticos
	metadataOffset := partitionStart + 2048
	metadataSize := int64(1048576) // 1MB completo de metadatos
	fmt.Printf("DEBUG: DESTRUCCIÓN TOTAL - Limpiando metadatos desde posición %d, tamaño %d bytes\n", metadataOffset, metadataSize)
	err = clearArea(diskFile, metadataOffset, metadataSize)
	if err != nil {
		fmt.Printf("ADVERTENCIA: No se pudo destruir metadatos: %v\n", err)
	} else {
		fmt.Println("✓ METADATOS COMPLETAMENTE DESTRUIDOS (bitmaps + inodos + grupos)")
	}

	// 4. Limpiar MASIVAMENTE los bloques de datos iniciales
	dataOffset := partitionStart + 1050624 // Después de metadatos
	dataSize := int64(2097152)             // 2MB de bloques de datos
	fmt.Printf("DEBUG: DESTRUCCIÓN TOTAL - Limpiando bloques de datos desde posición %d, tamaño %d bytes\n", dataOffset, dataSize)
	err = clearArea(diskFile, dataOffset, dataSize)
	if err != nil {
		fmt.Printf("ADVERTENCIA: No se pudo destruir bloques de datos: %v\n", err)
	} else {
		fmt.Println("✓ BLOQUES DE DATOS MASIVAMENTE DESTRUIDOS")
	}

	// 5. NUEVO: Patrón de destrucción disperso - limpiar bloques distribuidos
	fmt.Println("DEBUG: APLICANDO PATRÓN DE DESTRUCCIÓN DISPERSO...")
	disperseCleanCount := 0
	interval := int64(8192) // Cada 8KB
	for offset := partitionStart; offset < partitionStart+partitionSize && disperseCleanCount < 200; offset += interval {
		err = clearBlock(diskFile, offset, 1024)
		if err == nil {
			disperseCleanCount++
		}
		// También limpiar algunos bloques aleatorios cerca
		if disperseCleanCount%10 == 0 {
			clearBlock(diskFile, offset+1024, 1024)
			clearBlock(diskFile, offset+2048, 1024)
		}
	}
	fmt.Printf("✓ PATRÓN DISPERSO APLICADO (%d bloques destruidos)\n", disperseCleanCount)

	// 6. FINAL: Múltiples sincronizaciones para asegurar escritura
	for i := 0; i < 3; i++ {
		err = diskFile.Sync()
		if err != nil {
			fmt.Printf("ADVERTENCIA: Sincronización %d falló: %v\n", i+1, err)
		}
	}
	fmt.Println("✓ CAMBIOS FORZOSAMENTE SINCRONIZADOS AL DISCO")

	fmt.Println("==========================================")
	fmt.Println("DESTRUCCIÓN COMPLETA DEL SISTEMA EXT3 FINALIZADA")
	fmt.Println("RESULTADO: El sistema de archivos está COMPLETAMENTE DESTRUIDO")
	fmt.Println("==========================================")
	return nil
}

// clearBlock limpia un bloque específico escribiendo ceros
func clearBlock(diskFile *os.File, offset int64, size int64) error {
	zeroBlock := make([]byte, size)
	_, err := diskFile.WriteAt(zeroBlock, offset)
	return err
}

// clearArea limpia un área específica escribiendo ceros
func clearArea(diskFile *os.File, offset int64, size int64) error {
	bufferSize := int64(4096) // Buffer de 4KB para optimizar escritura
	buffer := make([]byte, bufferSize)

	for written := int64(0); written < size; written += bufferSize {
		remaining := size - written
		if remaining < bufferSize {
			buffer = make([]byte, remaining)
		}

		_, err := diskFile.WriteAt(buffer, offset+written)
		if err != nil {
			return err
		}
	}
	return nil
}
