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

	// LIMPIEZA SELECTIVA DEL SISTEMA DE ARCHIVOS EXT3 (no tan agresiva)
	blockSize := int64(1024) // Tamaño estándar de bloque EXT3

	// 1. Corromper el superbloque principal (pero no los backups)
	superblockOffset := partitionStart + 1024
	fmt.Printf("DEBUG: CORRUPCIÓN SELECTIVA - Corrompiendo superbloque principal en posición %d\n", superblockOffset)
	err = corruptArea(diskFile, superblockOffset, blockSize) // Solo 1KB del superbloque
	if err != nil {
		fmt.Printf("ADVERTENCIA: No se pudo corromper superbloque: %v\n", err)
	} else {
		fmt.Println("✓ Superbloque principal corrompido")
	}

	// 2. Corromper parcialmente los bitmaps (no todo)
	bitmapOffset := partitionStart + 2048
	bitmapSize := int64(4096) // Solo 4KB de bitmaps
	fmt.Printf("DEBUG: CORRUPCIÓN SELECTIVA - Corrompiendo bitmaps desde posición %d, tamaño %d bytes\n", bitmapOffset, bitmapSize)
	err = corruptArea(diskFile, bitmapOffset, bitmapSize)
	if err != nil {
		fmt.Printf("ADVERTENCIA: No se pudo corromper bitmaps: %v\n", err)
	} else {
		fmt.Println("✓ Bitmaps parcialmente corrompidos")
	}

	// 3. Corromper algunos inodos críticos (no todos)
	inodeOffset := partitionStart + 8192
	inodeSize := int64(2048) // Solo 2KB de inodos
	fmt.Printf("DEBUG: CORRUPCIÓN SELECTIVA - Corrompiendo inodos desde posición %d, tamaño %d bytes\n", inodeOffset, inodeSize)
	err = corruptArea(diskFile, inodeOffset, inodeSize)
	if err != nil {
		fmt.Printf("ADVERTENCIA: No se pudo corromper inodos: %v\n", err)
	} else {
		fmt.Println("✓ Inodos críticos corrompidos")
	}

	// 4. Corromper algunos bloques de datos (patrón disperso)
	fmt.Println("DEBUG: APLICANDO PATRÓN DE CORRUPCIÓN DISPERSO...")
	corruptCount := 0
	interval := int64(16384) // Cada 16KB
	for offset := partitionStart + 12288; offset < partitionStart+partitionSize && corruptCount < 50; offset += interval {
		err = corruptBlock(diskFile, offset, 1024)
		if err == nil {
			corruptCount++
		}
	}
	fmt.Printf("✓ PATRÓN DISPERSO APLICADO (%d bloques corrompidos)\n", corruptCount)

	// 5. Sincronizar cambios
	err = diskFile.Sync()
	if err != nil {
		fmt.Printf("ADVERTENCIA: Sincronización falló: %v\n", err)
	} else {
		fmt.Println("✓ CAMBIOS SINCRONIZADOS AL DISCO")
	}

	fmt.Println("==========================================")
	fmt.Println("CORRUPCIÓN SELECTIVA DEL SISTEMA EXT3 FINALIZADA")
	fmt.Println("RESULTADO: El sistema de archivos tiene corrupción parcial")
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

// corruptBlock corrompe un bloque específico con datos aleatorios
func corruptBlock(diskFile *os.File, offset int64, size int64) error {
	// Crear un patrón de corrupción (no completamente aleatorio para ser reproducible)
	corruptData := make([]byte, size)
	for i := range corruptData {
		corruptData[i] = byte(0xFF) // Patrón de bits 11111111
		if i%3 == 0 {
			corruptData[i] = byte(0xAA) // Patrón alternado
		} else if i%5 == 0 {
			corruptData[i] = byte(0x55) // Otro patrón alternado
		}
	}
	_, err := diskFile.WriteAt(corruptData, offset)
	return err
}

// corruptArea corrompe un área específica con datos parcialmente aleatorios
func corruptArea(diskFile *os.File, offset int64, size int64) error {
	bufferSize := int64(1024) // Buffer de 1KB para optimizar escritura

	for written := int64(0); written < size; written += bufferSize {
		remaining := size - written
		if remaining < bufferSize {
			bufferSize = remaining
		}

		// Crear datos de corrupción para este buffer
		corruptData := make([]byte, bufferSize)
		for i := range corruptData {
			// Mezcla de patrones para simular corrupción real
			switch (written + int64(i)) % 4 {
			case 0:
				corruptData[i] = byte(0xFF)
			case 1:
				corruptData[i] = byte(0x00)
			case 2:
				corruptData[i] = byte(0xAA)
			case 3:
				corruptData[i] = byte(0x55)
			}
		}

		_, err := diskFile.WriteAt(corruptData, offset+written)
		if err != nil {
			return err
		}
	}
	return nil
}
