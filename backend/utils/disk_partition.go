package utils

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"main/structs"
	"os"
	"strings"
)

const (
	PRIMARY  byte = 'P'
	EXTENDED byte = 'E'
	LOGICAL  byte = 'L'
)
const (
	BEST_FIT  = "BF"
	FIRST_FIT = "FF"
	WORST_FIT = "WF"
)

// calculateAvailableSpace calcula el espacio disponible en el disco
func calculateAvailableSpace(mbr *structs.MBR) int32 {
	diskSize := mbr.Mbr_size
	const MBR_SIZE = 47 + (4 * 59) // MBR + espacio para 4 particiones

	usedSpace := int32(MBR_SIZE)

	// Sumar el espacio usado por particiones primarias y extendidas
	for i := 0; i < 4; i++ {
		if mbr.Partitions[i].Part_status[0] == 1 {
			usedSpace += mbr.Partitions[i].Part_size
		}
	}

	return diskSize - usedSpace
}

// calculateAvailableSpaceInExtended calcula el espacio disponible en una partición extendida
func calculateAvailableSpaceInExtended(diskFile *os.File, extendedPartition *structs.PARTITION) (int32, error) {
	if extendedPartition == nil {
		return 0, errors.New("partición extendida no válida")
	}

	extendedStart := extendedPartition.Part_start
	extendedSize := extendedPartition.Part_size
	usedSpace := int32(0)

	// Recorrer todas las particiones lógicas para calcular el espacio usado
	currentPos := extendedStart
	ebrSize := int32(30)

	for {
		// Leer el EBR en la posición actual
		ebrData := make([]byte, ebrSize)
		_, err := diskFile.ReadAt(ebrData, int64(currentPos))
		if err != nil {
			break
		}

		// Verificar si hay una partición lógica en esta posición
		partSize := int32(ebrData[6]) | int32(ebrData[7])<<8 | int32(ebrData[8])<<16 | int32(ebrData[9])<<24
		nextEBR := int32(ebrData[10]) | int32(ebrData[11])<<8 | int32(ebrData[12])<<16 | int32(ebrData[13])<<24

		if partSize == 0 {
			break
		}

		// Sumar el espacio usado (EBR + partición)
		usedSpace += ebrSize + partSize

		// Si no hay siguiente EBR, terminar
		if nextEBR == 0 {
			break
		}

		// Ir al siguiente EBR
		currentPos = extendedStart + nextEBR
	}

	return extendedSize - usedSpace, nil
}

func CreatePartition(params *structs.FdiskParams) error {
	diskFile, err := os.OpenFile(params.Path, os.O_RDWR, 0666)
	if err != nil {
		return fmt.Errorf("error al abrir el disco: %v", err)
	}
	defer diskFile.Close()

	const MBR_SIZE = 47

	mbrBytes := make([]byte, MBR_SIZE)

	_, err = diskFile.ReadAt(mbrBytes, 0)
	if err != nil {
		return fmt.Errorf("error al leer el MBR: %v", err)
	}

	var mbr structs.MBR
	buffer := bytes.NewReader(mbrBytes)

	binary.Read(buffer, binary.LittleEndian, &mbr.Mbr_size)
	buffer.Read(mbr.Mbr_timestamp[:])
	buffer.Read(mbr.Mbr_uuid[:])
	binary.Read(buffer, binary.LittleEndian, &mbr.Mbr_partitions)
	buffer.Read(mbr.Dsk_fit[:])

	partitionDataOffset := int64(MBR_SIZE)
	partitionSize := int64(59) // Tamaño de cada estructura PARTITION

	for i := 0; i < 4; i++ {
		partPos := partitionDataOffset + int64(i)*partitionSize

		partData := make([]byte, partitionSize)
		_, err = diskFile.ReadAt(partData, partPos)
		if err != nil {
			return fmt.Errorf("error al leer la partición %d: %v", i, err)
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

		if mbr.Partitions[i].Part_status[0] == 1 {
			partName := string(bytes.Trim(mbr.Partitions[i].Part_name[:], "\x00"))
			if strings.EqualFold(partName, params.Name) {
				return fmt.Errorf("ya existe una partición con el nombre '%s'", params.Name)
			}
		}
	}

	// Calcular el tamaño solicitado en bytes
	sizeInBytes := params.GetSizeInBytes()

	// Verificar espacio disponible según el tipo de partición
	if params.Type == LOGICAL {
		// Para particiones lógicas, verificar espacio en la partición extendida
		var extendedPartition *structs.PARTITION
		for i := 0; i < 4; i++ {
			if mbr.Partitions[i].Part_status[0] == 1 && mbr.Partitions[i].Part_type[0] == EXTENDED {
				extendedPartition = &mbr.Partitions[i]
				break
			}
		}

		if extendedPartition == nil {
			return errors.New("no se puede crear una partición lógica sin una partición extendida")
		}

		availableSpace, err := calculateAvailableSpaceInExtended(diskFile, extendedPartition)
		if err != nil {
			return fmt.Errorf("error al calcular espacio disponible en partición extendida: %v", err)
		}

		// Necesitamos espacio para el EBR (30 bytes) + la partición
		requiredSpace := int32(30) + sizeInBytes
		if availableSpace < requiredSpace {
			return fmt.Errorf("no hay espacio suficiente en la partición extendida. Disponible: %d bytes, Requerido: %d bytes", availableSpace, requiredSpace)
		}
	} else {
		// Para particiones primarias y extendidas, verificar espacio total del disco
		availableSpace := calculateAvailableSpace(&mbr)
		if availableSpace < sizeInBytes {
			return fmt.Errorf("no hay espacio suficiente en el disco. Disponible: %d bytes, Requerido: %d bytes", availableSpace, sizeInBytes)
		}
	}

	if params.Type == EXTENDED {
		for i := 0; i < 4; i++ {
			if mbr.Partitions[i].Part_status[0] == 1 && mbr.Partitions[i].Part_type[0] == EXTENDED {
				return errors.New("ya existe una partición extendida en el disco")
			}
		}
	} else if params.Type == LOGICAL {
		extendedExists := false
		for i := 0; i < 4; i++ {
			if mbr.Partitions[i].Part_status[0] == 1 && mbr.Partitions[i].Part_type[0] == EXTENDED {
				extendedExists = true
				break
			}
		}
		if !extendedExists {
			return errors.New("no se puede crear una partición lógica sin una partición extendida")
		}
	}

	if params.Type != LOGICAL && mbr.Mbr_partitions >= 4 {
		return errors.New("ya se alcanzó el número máximo de particiones primarias y extendidas (4)")
	}

	if params.Type == LOGICAL {
		// Implementación de particiones lógicas
		return createLogicalPartition(diskFile, &mbr, params, sizeInBytes)
	} else {
		var newPartIndex int = -1
		for i := 0; i < 4; i++ {
			if mbr.Partitions[i].Part_status[0] == 0 {
				newPartIndex = i
				break
			}
		}

		if newPartIndex == -1 {
			return errors.New("no hay slots libres para crear una nueva partición")
		}

		mbr.Partitions[newPartIndex].Part_status[0] = 1
		mbr.Partitions[newPartIndex].Part_type[0] = params.Type
		mbr.Partitions[newPartIndex].Part_size = sizeInBytes

		switch params.Fit {
		case BEST_FIT:
			mbr.Partitions[newPartIndex].Part_fit[0] = 'B'
		case WORST_FIT:
			mbr.Partitions[newPartIndex].Part_fit[0] = 'W'
		default:
			mbr.Partitions[newPartIndex].Part_fit[0] = 'F'
		}

		copy(mbr.Partitions[newPartIndex].Part_name[:], params.Name)

		uuidBytes := GenerateUUID()
		copy(mbr.Partitions[newPartIndex].Part_uuid[:], uuidBytes[:16])

		mbr.Partitions[newPartIndex].Part_correlative = mbr.Mbr_partitions + 1
		mbr.Mbr_partitions++

		// Buscar la siguiente posición disponible después de las particiones existentes
		nextAvailableStart := int32(MBR_SIZE)
		for i := 0; i < 4; i++ {
			if mbr.Partitions[i].Part_status[0] != 0 && mbr.Partitions[i].Part_size > 0 {
				partEnd := mbr.Partitions[i].Part_start + mbr.Partitions[i].Part_size
				if partEnd > nextAvailableStart {
					nextAvailableStart = partEnd
				}
			}
		}

		mbr.Partitions[newPartIndex].Part_start = nextAvailableStart
		var mbrBuffer bytes.Buffer
		binary.Write(&mbrBuffer, binary.LittleEndian, mbr.Mbr_size)
		mbrBuffer.Write(mbr.Mbr_timestamp[:])
		mbrBuffer.Write(mbr.Mbr_uuid[:])
		binary.Write(&mbrBuffer, binary.LittleEndian, mbr.Mbr_partitions)
		mbrBuffer.Write(mbr.Dsk_fit[:])

		_, err = diskFile.WriteAt(mbrBuffer.Bytes(), 0)
		if err != nil {
			return fmt.Errorf("error al actualizar el MBR: %v", err)
		}

		partPos := partitionDataOffset + int64(newPartIndex)*partitionSize
		var partBuffer bytes.Buffer

		partBuffer.Write(mbr.Partitions[newPartIndex].Part_status[:])
		partBuffer.Write(mbr.Partitions[newPartIndex].Part_type[:])
		partBuffer.Write(mbr.Partitions[newPartIndex].Part_fit[:])
		binary.Write(&partBuffer, binary.LittleEndian, mbr.Partitions[newPartIndex].Part_start)
		binary.Write(&partBuffer, binary.LittleEndian, mbr.Partitions[newPartIndex].Part_size)
		partBuffer.Write(mbr.Partitions[newPartIndex].Part_name[:])
		partBuffer.Write(mbr.Partitions[newPartIndex].Part_uuid[:])
		binary.Write(&partBuffer, binary.LittleEndian, mbr.Partitions[newPartIndex].Part_correlative)

		_, err = diskFile.WriteAt(partBuffer.Bytes(), partPos)
		if err != nil {
			return fmt.Errorf("error al escribir la partición: %v", err)
		}

		// CRÍTICO: Si es una partición extendida, crear el primer EBR vacío
		if params.Type == EXTENDED {
			err = createInitialEBR(diskFile, mbr.Partitions[newPartIndex].Part_start)
			if err != nil {
				return fmt.Errorf("error al crear el EBR inicial: %v", err)
			}
		}
	}

	fmt.Printf("Partición '%s' creada exitosamente en el disco '%s'\n", params.Name, params.Path)
	return nil
}

// createLogicalPartition crea una partición lógica dentro de una partición extendida
func createLogicalPartition(diskFile *os.File, mbr *structs.MBR, params *structs.FdiskParams, sizeInBytes int32) error {
	// Buscar la partición extendida
	var extendedPartition *structs.PARTITION
	for i := 0; i < 4; i++ {
		if mbr.Partitions[i].Part_status[0] == 1 && mbr.Partitions[i].Part_type[0] == EXTENDED {
			extendedPartition = &mbr.Partitions[i]
			break
		}
	}

	if extendedPartition == nil {
		return errors.New("no se encontró una partición extendida para crear la partición lógica")
	}

	// Verificar que no existe una partición con el mismo nombre (primarias/extendidas)
	for i := 0; i < 4; i++ {
		if mbr.Partitions[i].Part_status[0] == 1 {
			partName := string(bytes.Trim(mbr.Partitions[i].Part_name[:], "\x00"))
			if strings.EqualFold(partName, params.Name) {
				return fmt.Errorf("ya existe una partición con el nombre '%s'", params.Name)
			}
		}
	}

	// Calcular el espacio disponible en la partición extendida
	extendedStart := extendedPartition.Part_start

	// Buscar la posición donde crear la nueva partición lógica
	currentPos := extendedStart
	ebrSize := int32(30)    // Tamaño del EBR
	lastEBRPos := int32(-1) // Posición del último EBR para actualizar su Part_next

	// Leer todas las particiones lógicas existentes para encontrar el último EBR
	for {
		// Leer el EBR en la posición actual
		ebrData := make([]byte, ebrSize)
		_, err := diskFile.ReadAt(ebrData, int64(currentPos))
		if err != nil {
			break
		}

		// Verificar si hay una partición lógica en esta posición
		partSize := int32(ebrData[6]) | int32(ebrData[7])<<8 | int32(ebrData[8])<<16 | int32(ebrData[9])<<24
		nextEBR := int32(ebrData[10]) | int32(ebrData[11])<<8 | int32(ebrData[12])<<16 | int32(ebrData[13])<<24

		// Si no hay partición aquí, podemos usar este espacio
		if partSize == 0 {
			break
		}

		// Verificar duplicado de nombre
		partName := string(bytes.Trim(ebrData[14:30], "\x00"))
		if strings.EqualFold(partName, params.Name) {
			return fmt.Errorf("ya existe una partición lógica con el nombre '%s'", params.Name)
		}

		// Si no hay siguiente EBR, podemos crear después de esta partición
		if nextEBR == 0 {
			lastEBRPos = currentPos // Guardar la posición del último EBR
			currentPos = currentPos + ebrSize + partSize
			break
		}

		// Ir al siguiente EBR
		currentPos = extendedStart + nextEBR
	}

	// Crear el nuevo EBR
	var ebr structs.EBR
	ebr.Part_mount[0] = 0 // No montado

	switch params.Fit {
	case BEST_FIT:
		ebr.Part_fit[0] = 'B'
	case WORST_FIT:
		ebr.Part_fit[0] = 'W'
	default:
		ebr.Part_fit[0] = 'F'
	}

	ebr.Part_start = ebrSize // La partición empieza después del EBR
	ebr.Part_size = sizeInBytes
	ebr.Part_next = 0 // Por ahora no hay siguiente (se actualizará si se crea otra)
	copy(ebr.Part_name[:], params.Name)

	// Escribir el EBR al disco
	var ebrBuffer bytes.Buffer
	ebrBuffer.Write(ebr.Part_mount[:])
	ebrBuffer.Write(ebr.Part_fit[:])
	binary.Write(&ebrBuffer, binary.LittleEndian, ebr.Part_start)
	binary.Write(&ebrBuffer, binary.LittleEndian, ebr.Part_size)
	binary.Write(&ebrBuffer, binary.LittleEndian, ebr.Part_next)
	ebrBuffer.Write(ebr.Part_name[:])

	_, err := diskFile.WriteAt(ebrBuffer.Bytes(), int64(currentPos))
	if err != nil {
		return fmt.Errorf("error al escribir el EBR: %v", err)
	}

	// CRÍTICO: Actualizar el EBR anterior para que apunte al nuevo EBR
	if lastEBRPos != -1 {
		// Calcular la posición relativa del nuevo EBR desde el inicio de la partición extendida
		newEBRRelativePos := currentPos - extendedStart

		// Leer el EBR anterior
		lastEBRData := make([]byte, ebrSize)
		_, err := diskFile.ReadAt(lastEBRData, int64(lastEBRPos))
		if err != nil {
			return fmt.Errorf("error al leer el EBR anterior: %v", err)
		}

		// Actualizar el campo Part_next del EBR anterior
		var lastEBRBuffer bytes.Buffer
		lastEBRBuffer.Write(lastEBRData[:10])                                // Part_mount, Part_fit, Part_start, Part_size (10 bytes)
		binary.Write(&lastEBRBuffer, binary.LittleEndian, newEBRRelativePos) // Part_next (4 bytes)
		lastEBRBuffer.Write(lastEBRData[14:])                                // Part_name (16 bytes)

		// Escribir el EBR anterior actualizado
		_, err = diskFile.WriteAt(lastEBRBuffer.Bytes(), int64(lastEBRPos))
		if err != nil {
			return fmt.Errorf("error al actualizar el EBR anterior: %v", err)
		}
	}

	// No incrementamos mbr.Mbr_partitions para particiones lógicas
	// ya que este contador es solo para particiones primarias y extendidas

	return nil
}

// createInitialEBR crea el primer EBR vacío cuando se crea una partición extendida
func createInitialEBR(diskFile *os.File, extendedStart int32) error {
	// Crear un EBR vacío
	var ebr structs.EBR

	// Inicializar todos los campos en cero/vacío
	ebr.Part_mount[0] = 0 // No montado
	ebr.Part_fit[0] = 0   // Sin ajuste definido
	ebr.Part_start = 0    // Sin partición
	ebr.Part_size = 0     // Sin tamaño
	ebr.Part_next = 0     // Sin siguiente EBR
	// ebr.Part_name ya está inicializado en ceros

	// Escribir el EBR vacío al inicio de la partición extendida
	var ebrBuffer bytes.Buffer
	ebrBuffer.Write(ebr.Part_mount[:])
	ebrBuffer.Write(ebr.Part_fit[:])
	binary.Write(&ebrBuffer, binary.LittleEndian, ebr.Part_start)
	binary.Write(&ebrBuffer, binary.LittleEndian, ebr.Part_size)
	binary.Write(&ebrBuffer, binary.LittleEndian, ebr.Part_next)
	ebrBuffer.Write(ebr.Part_name[:])

	_, err := diskFile.WriteAt(ebrBuffer.Bytes(), int64(extendedStart))
	if err != nil {
		return fmt.Errorf("error al escribir el EBR inicial: %v", err)
	}

	fmt.Printf("EBR inicial creado en la posición %d\n", extendedStart)
	return nil
}

// DeletePartition elimina una partición del disco
func DeletePartition(params *structs.FdiskParams) error {
	// Verificar que el disco existe
	if _, err := os.Stat(params.Path); os.IsNotExist(err) {
		return fmt.Errorf("el disco no existe: %s", params.Path)
	}

	// Leer el MBR del disco
	mbr, err := ReadMBR(params.Path)
	if err != nil {
		return fmt.Errorf("error al leer MBR: %v", err)
	}

	// Buscar la partición en particiones primarias y extendidas
	partitionIndex := -1
	var targetPartition *structs.PARTITION

	for i := 0; i < 4; i++ {
		partition := mbr.Partitions[i]

		// Verificar si la partición está en uso (Part_status[0] != 0)
		if partition.Part_status[0] != 0 {
			// Limpiar el nombre de la partición (quitar bytes nulos)
			partitionName := strings.TrimRight(string(partition.Part_name[:]), "\x00")

			if strings.EqualFold(partitionName, params.Name) {
				partitionIndex = i
				targetPartition = &mbr.Partitions[i]
				break
			}
		}
	}

	// Si no se encontró en particiones primarias/extendidas, buscar en lógicas
	if partitionIndex == -1 {
		for i := 0; i < 4; i++ {
			partition := mbr.Partitions[i]

			// Verificar si es una partición extendida
			if partition.Part_status[0] != 0 && partition.Part_type[0] == 'E' {
				found, err := deleteLogicalPartitionByName(params.Path, partition.Part_start, params.Name, params.Delete)
				if err != nil {
					return fmt.Errorf("error al buscar en particiones lógicas: %v", err)
				}
				if found {
					fmt.Printf("Partición lógica '%s' eliminada exitosamente (%s)\n", params.Name, params.Delete)
					return nil
				}
			}
		}
	}

	if partitionIndex == -1 {
		return fmt.Errorf("la partición '%s' no existe", params.Name)
	}

	// Aquí ya tenemos la partición encontrada en particiones primarias/extendidas
	// Abrir el archivo para modificarlo
	diskFile, err := os.OpenFile(params.Path, os.O_RDWR, 0666)
	if err != nil {
		return fmt.Errorf("error al abrir el disco: %v", err)
	}
	defer diskFile.Close()

	// Si es una partición extendida, eliminar todas las lógicas dentro
	if targetPartition.Part_type[0] == 'E' {
		err = deleteLogicalPartitions(diskFile, targetPartition, params.Delete)
		if err != nil {
			return fmt.Errorf("error al eliminar particiones lógicas: %v", err)
		}
	}

	// Eliminar la partición según el tipo (fast o full)
	if params.Delete == "full" {
		// Rellenar el espacio con \0
		err = fillPartitionWithZeros(diskFile, targetPartition)
		if err != nil {
			return fmt.Errorf("error al limpiar la partición: %v", err)
		}
	}

	// Marcar la partición como vacía en la tabla de particiones
	mbr.Partitions[partitionIndex] = structs.PARTITION{} // Estructura vacía

	// Escribir el MBR actualizado
	err = writeMBR(diskFile, mbr)
	if err != nil {
		return fmt.Errorf("error al escribir el MBR: %v", err)
	}

	fmt.Printf("Partición '%s' eliminada exitosamente (%s)\n", params.Name, params.Delete)
	return nil
}

// ModifyPartitionSize modifica el tamaño de una partición existente
func ModifyPartitionSize(params *structs.FdiskParams) error {
	// Verificar que el disco exists
	if _, err := os.Stat(params.Path); os.IsNotExist(err) {
		return fmt.Errorf("el disco no existe: %s", params.Path)
	}

	// Leer el MBR del disco
	mbr, err := ReadMBR(params.Path)
	if err != nil {
		return fmt.Errorf("error al leer MBR: %v", err)
	}

	// Buscar la partición en particiones primarias y extendidas
	partitionIndex := -1
	var targetPartition *structs.PARTITION

	for i := 0; i < 4; i++ {
		partition := mbr.Partitions[i]

		// Verificar si la partición está en uso (Part_status[0] != 0)
		if partition.Part_status[0] != 0 {
			// Limpiar el nombre de la partición (quitar bytes nulos)
			partitionName := strings.TrimRight(string(partition.Part_name[:]), "\x00")

			if strings.EqualFold(partitionName, params.Name) {
				partitionIndex = i
				targetPartition = &mbr.Partitions[i]
				break
			}
		}
	}

	// Si no se encontró en particiones primarias/extendidas, buscar en lógicas
	if partitionIndex == -1 {
		for i := 0; i < 4; i++ {
			partition := mbr.Partitions[i]

			// Verificar si es una partición extendida
			if partition.Part_status[0] != 0 && partition.Part_type[0] == 'E' {
				found, err := modifyLogicalPartitionSize(params.Path, partition.Part_start, params.Name, params.GetAddSizeInBytes())
				if err != nil {
					return fmt.Errorf("error al modificar partición lógica: %v", err)
				}
				if found {
					fmt.Printf("Partición lógica '%s' modificada exitosamente\n", params.Name)
					return nil
				}
			}
		}
	}

	if partitionIndex == -1 {
		return fmt.Errorf("la partición '%s' no existe", params.Name)
	}

	// Aquí ya tenemos la partición encontrada
	// Abrir el archivo para modificarlo
	diskFile, err := os.OpenFile(params.Path, os.O_RDWR, 0666)
	if err != nil {
		return fmt.Errorf("error al abrir el disco: %v", err)
	}
	defer diskFile.Close()

	addSizeInBytes := params.GetAddSizeInBytes()

	// Validar que no se reduzca a tamaño negativo
	if addSizeInBytes < 0 && targetPartition.Part_size+addSizeInBytes <= 0 {
		return fmt.Errorf("no se puede reducir la partición: el tamaño resultante sería negativo o cero")
	}

	// Si se está agregando espacio, verificar que hay espacio disponible después de la partición
	if addSizeInBytes > 0 {
		err = validateSpaceAfterPartition(mbr, targetPartition, addSizeInBytes)
		if err != nil {
			return fmt.Errorf("no hay espacio suficiente: %v", err)
		}
	}

	// Modificar el tamaño
	newSize := targetPartition.Part_size + addSizeInBytes
	mbr.Partitions[partitionIndex].Part_size = newSize

	// Escribir el MBR actualizado
	err = writeMBR(diskFile, mbr)
	if err != nil {
		return fmt.Errorf("error al escribir el MBR: %v", err)
	}

	operation := "agregado"
	if addSizeInBytes < 0 {
		operation = "quitado"
		addSizeInBytes = -addSizeInBytes
	}

	fmt.Printf("Espacio %s exitosamente a la partición '%s': %d bytes\n", operation, params.Name, addSizeInBytes)
	return nil
}

// Funciones auxiliares

// readMBR lee el MBR completo del disco
func readMBR(diskFile *os.File) (structs.MBR, error) {
	var mbr structs.MBR
	const MBR_SIZE = 47

	mbrBytes := make([]byte, MBR_SIZE)
	_, err := diskFile.ReadAt(mbrBytes, 0)
	if err != nil {
		return mbr, err
	}

	buffer := bytes.NewReader(mbrBytes)
	binary.Read(buffer, binary.LittleEndian, &mbr.Mbr_size)
	buffer.Read(mbr.Mbr_timestamp[:])
	buffer.Read(mbr.Mbr_uuid[:])
	binary.Read(buffer, binary.LittleEndian, &mbr.Mbr_partitions)
	buffer.Read(mbr.Dsk_fit[:])

	// Leer las particiones
	partitionDataOffset := int64(MBR_SIZE)
	partitionSize := int64(59)

	for i := 0; i < 4; i++ {
		partPos := partitionDataOffset + int64(i)*partitionSize
		partData := make([]byte, partitionSize)
		_, err = diskFile.ReadAt(partData, partPos)
		if err != nil {
			return mbr, err
		}

		partBuffer := bytes.NewReader(partData)
		partBuffer.Read(mbr.Partitions[i].Part_status[:])
		partBuffer.Read(mbr.Partitions[i].Part_type[:])
		partBuffer.Read(mbr.Partitions[i].Part_fit[:])
		binary.Read(partBuffer, binary.LittleEndian, &mbr.Partitions[i].Part_start)
		binary.Read(partBuffer, binary.LittleEndian, &mbr.Partitions[i].Part_size)
		partBuffer.Read(mbr.Partitions[i].Part_name[:])
	}

	return mbr, nil
}

// writeMBR escribe el MBR completo al disco
func writeMBR(diskFile *os.File, mbr *structs.MBR) error {
	// Escribir la cabecera del MBR
	var mbrBuffer bytes.Buffer
	binary.Write(&mbrBuffer, binary.LittleEndian, mbr.Mbr_size)
	mbrBuffer.Write(mbr.Mbr_timestamp[:])
	mbrBuffer.Write(mbr.Mbr_uuid[:])
	binary.Write(&mbrBuffer, binary.LittleEndian, mbr.Mbr_partitions)
	mbrBuffer.Write(mbr.Dsk_fit[:])

	_, err := diskFile.WriteAt(mbrBuffer.Bytes(), 0)
	if err != nil {
		return err
	}

	// Escribir las particiones
	const MBR_SIZE = 47
	partitionDataOffset := int64(MBR_SIZE)
	partitionSize := int64(59)

	for i := 0; i < 4; i++ {
		var partBuffer bytes.Buffer
		partBuffer.Write(mbr.Partitions[i].Part_status[:])
		partBuffer.Write(mbr.Partitions[i].Part_type[:])
		partBuffer.Write(mbr.Partitions[i].Part_fit[:])
		binary.Write(&partBuffer, binary.LittleEndian, mbr.Partitions[i].Part_start)
		binary.Write(&partBuffer, binary.LittleEndian, mbr.Partitions[i].Part_size)
		partBuffer.Write(mbr.Partitions[i].Part_name[:])

		partPos := partitionDataOffset + int64(i)*partitionSize
		_, err = diskFile.WriteAt(partBuffer.Bytes(), partPos)
		if err != nil {
			return err
		}
	}

	return nil
}

// deleteLogicalPartitions elimina todas las particiones lógicas de una partición extendida
func deleteLogicalPartitions(diskFile *os.File, extendedPartition *structs.PARTITION, deleteType string) error {
	currentPos := extendedPartition.Part_start

	for currentPos != -1 {
		// Leer el EBR
		var ebr structs.EBR
		ebrBytes := make([]byte, 59) // Tamaño del EBR
		_, err := diskFile.ReadAt(ebrBytes, int64(currentPos))
		if err != nil {
			break
		}

		buffer := bytes.NewReader(ebrBytes)
		buffer.Read(ebr.Part_mount[:])
		buffer.Read(ebr.Part_fit[:])
		binary.Read(buffer, binary.LittleEndian, &ebr.Part_start)
		binary.Read(buffer, binary.LittleEndian, &ebr.Part_size)
		binary.Read(buffer, binary.LittleEndian, &ebr.Part_next)
		buffer.Read(ebr.Part_name[:])

		// Si es eliminación completa, limpiar el espacio
		if deleteType == "full" && ebr.Part_size > 0 {
			fillAreaWithZeros(diskFile, ebr.Part_start, ebr.Part_size)
		}

		// Limpiar el EBR
		emptyEBR := make([]byte, 59)
		_, err = diskFile.WriteAt(emptyEBR, int64(currentPos))
		if err != nil {
			return err
		}

		// Moverse al siguiente EBR
		currentPos = ebr.Part_next
	}

	return nil
}

// fillPartitionWithZeros rellena una partición con ceros
func fillPartitionWithZeros(diskFile *os.File, partition *structs.PARTITION) error {
	return fillAreaWithZeros(diskFile, partition.Part_start, partition.Part_size)
}

// fillAreaWithZeros rellena un área del disco con ceros
func fillAreaWithZeros(diskFile *os.File, start int32, size int32) error {
	const bufferSize = 1024 * 1024 // 1MB buffer
	zeros := make([]byte, bufferSize)

	remaining := size
	currentPos := start

	for remaining > 0 {
		writeSize := bufferSize
		if remaining < int32(bufferSize) {
			writeSize = int(remaining)
		}

		_, err := diskFile.WriteAt(zeros[:writeSize], int64(currentPos))
		if err != nil {
			return err
		}

		currentPos += int32(writeSize)
		remaining -= int32(writeSize)
	}

	return nil
}

// validateSpaceAfterPartition valida que hay espacio disponible después de una partición
func validateSpaceAfterPartition(mbr *structs.MBR, partition *structs.PARTITION, requiredSpace int32) error {
	partitionEnd := partition.Part_start + partition.Part_size
	diskEnd := mbr.Mbr_size

	// Buscar la siguiente partición más cercana
	nextPartitionStart := diskEnd
	for i := 0; i < 4; i++ {
		if mbr.Partitions[i].Part_status[0] == 1 &&
			mbr.Partitions[i].Part_start > partitionEnd &&
			mbr.Partitions[i].Part_start < nextPartitionStart {
			nextPartitionStart = mbr.Partitions[i].Part_start
		}
	}

	availableSpace := nextPartitionStart - partitionEnd

	if availableSpace < requiredSpace {
		return fmt.Errorf("espacio insuficiente después de la partición. Disponible: %d bytes, Requerido: %d bytes",
			availableSpace, requiredSpace)
	}

	return nil
}

// deleteLogicalPartitionByName busca y elimina una partición lógica por nombre
func deleteLogicalPartitionByName(diskPath string, extendedStart int32, partitionName string, deleteType string) (bool, error) {
	diskFile, err := os.OpenFile(diskPath, os.O_RDWR, 0666)
	if err != nil {
		return false, fmt.Errorf("error al abrir disco: %v", err)
	}
	defer diskFile.Close()

	currentEBRPos := extendedStart

	for currentEBRPos != 0 {
		var ebr structs.EBR
		err := ReadEBR(diskFile, &ebr, int64(currentEBRPos))
		if err != nil {
			break
		}

		// Verificar si el EBR contiene una partición válida
		if ebr.Part_mount[0] != 0 {
			// Limpiar el nombre de la partición
			currentPartName := strings.TrimRight(string(ebr.Part_name[:]), "\x00")

			if strings.EqualFold(currentPartName, partitionName) {
				// Encontramos la partición a eliminar
				if deleteType == "full" {
					// Llenar la partición con ceros
					zeros := make([]byte, 1024) // Buffer de 1KB
					for pos := int64(ebr.Part_start); pos < int64(ebr.Part_start+ebr.Part_size); pos += int64(len(zeros)) {
						remaining := int64(ebr.Part_start+ebr.Part_size) - pos
						if remaining < int64(len(zeros)) {
							zeros = make([]byte, remaining)
						}
						_, err := diskFile.WriteAt(zeros, pos)
						if err != nil {
							return false, fmt.Errorf("error al llenar partición con ceros: %v", err)
						}
					}
				}

				// Limpiar el EBR (marcarlo como vacío)
				emptyEBR := make([]byte, 30) // Tamaño del EBR
				if ebr.Part_next != 0 {
					// Mantener el enlace al siguiente EBR
					emptyEBR[10] = byte(ebr.Part_next)
					emptyEBR[11] = byte(ebr.Part_next >> 8)
					emptyEBR[12] = byte(ebr.Part_next >> 16)
					emptyEBR[13] = byte(ebr.Part_next >> 24)
				}

				// Escribir el EBR vacío
				_, err = diskFile.WriteAt(emptyEBR, int64(currentEBRPos))
				if err != nil {
					return false, fmt.Errorf("error al escribir EBR vacío: %v", err)
				}

				return true, nil
			}
		}

		// Ir al siguiente EBR
		currentEBRPos = ebr.Part_next
	}

	return false, nil // No se encontró la partición
}

// modifyLogicalPartitionSize busca y modifica el tamaño de una partición lógica por nombre
func modifyLogicalPartitionSize(diskPath string, extendedStart int32, partitionName string, addSizeInBytes int32) (bool, error) {
	diskFile, err := os.OpenFile(diskPath, os.O_RDWR, 0666)
	if err != nil {
		return false, fmt.Errorf("error al abrir disco: %v", err)
	}
	defer diskFile.Close()

	currentEBRPos := extendedStart

	for currentEBRPos != 0 {
		var ebr structs.EBR
		err := ReadEBR(diskFile, &ebr, int64(currentEBRPos))
		if err != nil {
			break
		}

		// Verificar si el EBR contiene una partición válida
		if ebr.Part_mount[0] != 0 {
			// Limpiar el nombre de la partición
			currentPartName := strings.TrimRight(string(ebr.Part_name[:]), "\x00")

			if strings.EqualFold(currentPartName, partitionName) {
				// Encontramos la partición a modificar
				newSize := ebr.Part_size + addSizeInBytes

				// Validar que el nuevo tamaño no sea negativo
				if newSize <= 0 {
					return false, fmt.Errorf("el nuevo tamaño de la partición sería negativo o cero")
				}

				// Actualizar el tamaño en el EBR
				ebr.Part_size = newSize

				// Escribir el EBR actualizado
				ebrData := make([]byte, 30) // Tamaño del EBR
				ebrData[0] = ebr.Part_mount[0]
				ebrData[1] = ebr.Part_fit[0]

				// Part_start (4 bytes, little endian)
				ebrData[2] = byte(ebr.Part_start)
				ebrData[3] = byte(ebr.Part_start >> 8)
				ebrData[4] = byte(ebr.Part_start >> 16)
				ebrData[5] = byte(ebr.Part_start >> 24)

				// Part_size (4 bytes, little endian)
				ebrData[6] = byte(ebr.Part_size)
				ebrData[7] = byte(ebr.Part_size >> 8)
				ebrData[8] = byte(ebr.Part_size >> 16)
				ebrData[9] = byte(ebr.Part_size >> 24)

				// Part_next (4 bytes, little endian)
				ebrData[10] = byte(ebr.Part_next)
				ebrData[11] = byte(ebr.Part_next >> 8)
				ebrData[12] = byte(ebr.Part_next >> 16)
				ebrData[13] = byte(ebr.Part_next >> 24)

				// Part_name (16 bytes)
				copy(ebrData[14:30], ebr.Part_name[:])

				_, err = diskFile.WriteAt(ebrData, int64(currentEBRPos))
				if err != nil {
					return false, fmt.Errorf("error al escribir EBR actualizado: %v", err)
				}

				return true, nil
			}
		}

		// Ir al siguiente EBR
		currentEBRPos = ebr.Part_next
	}

	return false, nil // No se encontró la partición
}

// Funciones auxiliares
