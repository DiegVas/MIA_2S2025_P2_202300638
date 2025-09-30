package diskmanager

import (
	"bufio"
	"fmt"
	"main/structs"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
)

// RegisterDisk registra un disco en el archivo ~/disks
func RegisterDisk(params *structs.MkdiskParams) error {
	// Obtener directorio home del usuario
	usr, err := user.Current()
	if err != nil {
		return fmt.Errorf("error al obtener directorio home: %v", err)
	}

	disksFilePath := filepath.Join(usr.HomeDir, "disks")

	// Leer contenido existente del archivo (si existe)
	var lines []string
	var diskExists bool
	var updatedLine string

	// Obtener el nombre del disco (basename del path)
	diskName := filepath.Base(params.Path)
	// Crear la línea que queremos escribir/actualizar (sin particiones inicialmente)
	newLine := fmt.Sprintf("%s|%s|%d|%s|", params.Path, params.Fit, params.Size, diskName)

	// Verificar si el archivo existe y leer su contenido
	if _, err := os.Stat(disksFilePath); err == nil {
		file, err := os.Open(disksFilePath)
		if err != nil {
			return fmt.Errorf("error al abrir archivo disks: %v", err)
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Text()
			if line == "" {
				continue
			}

			parts := strings.Split(line, "|")
			if len(parts) >= 1 && parts[0] == params.Path {
				// El disco ya existe, lo reemplazamos
				diskExists = true
				updatedLine = newLine
				lines = append(lines, updatedLine)
			} else {
				// Mantener la línea existente
				lines = append(lines, line)
			}
		}

		if err := scanner.Err(); err != nil {
			return fmt.Errorf("error al leer archivo disks: %v", err)
		}
	}

	// Si el disco no existe, agregarlo
	if !diskExists {
		lines = append(lines, newLine)
	}

	// Escribir el archivo actualizado
	file, err := os.Create(disksFilePath)
	if err != nil {
		return fmt.Errorf("error al crear/abrir archivo disks: %v", err)
	}
	defer file.Close()

	for _, line := range lines {
		_, err := file.WriteString(line + "\n")
		if err != nil {
			return fmt.Errorf("error al escribir en archivo disks: %v", err)
		}
	}

	return nil
}

// GetRegisteredDisks obtiene la lista de discos registrados
func GetRegisteredDisks() ([]map[string]interface{}, error) {
	usr, err := user.Current()
	if err != nil {
		return nil, fmt.Errorf("error al obtener directorio home: %v", err)
	}

	disksFilePath := filepath.Join(usr.HomeDir, "disks")

	var disks []map[string]interface{}

	// Verificar si el archivo existe
	if _, err := os.Stat(disksFilePath); os.IsNotExist(err) {
		// El archivo no existe, retornar lista vacía
		return disks, nil
	}

	file, err := os.Open(disksFilePath)
	if err != nil {
		return nil, fmt.Errorf("error al abrir archivo disks: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		parts := strings.Split(line, "|")
		if len(parts) >= 4 {
			size, err := strconv.Atoi(parts[2])
			if err != nil {
				continue // Saltar líneas mal formateadas
			}

			// Obtener particiones (pueden estar vacías)
			var partitions []string
			if len(parts) > 4 && parts[4] != "" {
				partitions = strings.Split(parts[4], ",")
			}

			disk := map[string]interface{}{
				"path":       parts[0],
				"fit":        parts[1],
				"size":       size,
				"name":       parts[3],
				"partitions": partitions,
			}
			disks = append(disks, disk)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error al leer archivo disks: %v", err)
	}

	return disks, nil
}

// UnregisterDisk elimina un disco del archivo ~/disks
func UnregisterDisk(diskPath string) error {
	// Obtener directorio home del usuario
	usr, err := user.Current()
	if err != nil {
		return fmt.Errorf("error al obtener directorio home: %v", err)
	}

	disksFilePath := filepath.Join(usr.HomeDir, "disks")

	// Verificar si el archivo existe
	if _, err := os.Stat(disksFilePath); os.IsNotExist(err) {
		// El archivo no existe, no hay nada que eliminar
		return nil
	}

	// Leer contenido existente del archivo
	var lines []string
	var diskFound bool

	file, err := os.Open(disksFilePath)
	if err != nil {
		return fmt.Errorf("error al abrir archivo disks: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		parts := strings.Split(line, "|")
		if len(parts) >= 1 && parts[0] == diskPath {
			// Este es el disco que queremos eliminar, no lo agregamos a lines
			diskFound = true
		} else {
			// Mantener la línea existente
			lines = append(lines, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error al leer archivo disks: %v", err)
	}

	// Si no se encontró el disco, no hay nada que hacer
	if !diskFound {
		return nil
	}

	// Escribir el archivo actualizado (sin el disco eliminado)
	file, err = os.Create(disksFilePath)
	if err != nil {
		return fmt.Errorf("error al crear/abrir archivo disks: %v", err)
	}
	defer file.Close()

	for _, line := range lines {
		_, err := file.WriteString(line + "\n")
		if err != nil {
			return fmt.Errorf("error al escribir en archivo disks: %v", err)
		}
	}

	return nil
}

// RegisterPartition agrega una partición a un disco existente en el registro
func RegisterPartition(diskPath string, partitionName string) error {
	// Obtener directorio home del usuario
	usr, err := user.Current()
	if err != nil {
		return fmt.Errorf("error al obtener directorio home: %v", err)
	}

	disksFilePath := filepath.Join(usr.HomeDir, "disks")

	// Verificar si el archivo existe
	if _, err := os.Stat(disksFilePath); os.IsNotExist(err) {
		return fmt.Errorf("archivo de discos no existe, debe crear el disco primero")
	}

	// Leer contenido existente del archivo
	var lines []string
	var diskFound bool

	file, err := os.Open(disksFilePath)
	if err != nil {
		return fmt.Errorf("error al abrir archivo disks: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		parts := strings.Split(line, "|")
		if len(parts) >= 4 && parts[0] == diskPath {
			// Este es el disco al que queremos agregar la partición
			diskFound = true

			// Obtener particiones existentes
			var existingPartitions []string
			if len(parts) > 4 && parts[4] != "" {
				existingPartitions = strings.Split(parts[4], ",")
			}

			// Verificar si la partición ya existe
			partitionExists := false
			for _, existing := range existingPartitions {
				if existing == partitionName {
					partitionExists = true
					break
				}
			}

			// Si la partición no existe, agregarla
			if !partitionExists {
				existingPartitions = append(existingPartitions, partitionName)
			}

			// Reconstruir la línea con las particiones actualizadas
			partitionsStr := strings.Join(existingPartitions, ",")
			updatedLine := fmt.Sprintf("%s|%s|%s|%s|%s", parts[0], parts[1], parts[2], parts[3], partitionsStr)
			lines = append(lines, updatedLine)
		} else {
			// Mantener la línea existente
			lines = append(lines, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error al leer archivo disks: %v", err)
	}

	// Si no se encontró el disco, retornar error
	if !diskFound {
		return fmt.Errorf("disco no encontrado en el registro: %s", diskPath)
	}

	// Escribir el archivo actualizado
	file, err = os.Create(disksFilePath)
	if err != nil {
		return fmt.Errorf("error al crear/abrir archivo disks: %v", err)
	}
	defer file.Close()

	for _, line := range lines {
		_, err := file.WriteString(line + "\n")
		if err != nil {
			return fmt.Errorf("error al escribir en archivo disks: %v", err)
		}
	}

	return nil
}
