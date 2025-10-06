package common

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"main/structs"
	"os"
	"strings"
)

func FindFileInodeByPath(file *os.File, path string, inodePos, blockPos int32) (int32, error) {

	fmt.Printf("DEBUG FindFileInodeByPath: Buscando path='%s', inodePos=%d, blockPos=%d\n", path, inodePos, blockPos)

	// Si el path es la raíz, retornar el inodo 0 directamente
	if path == "/" {
		return 0, nil
	}

	if path == "" {
		return 0, nil // Después de normalizar, si queda vacío, es la raíz
	}

	// Dividir el path en componentes y filtrar componentes vacíos
	components := strings.Split(path, "/")
	var filteredComponents []string
	for _, component := range components {
		if component != "" {
			filteredComponents = append(filteredComponents, component)
		}
	}

	fmt.Printf("DEBUG FindFileInodeByPath: Componentes filtrados: %v\n", filteredComponents)

	// Si no hay componentes válidos, es la raíz
	if len(filteredComponents) == 0 {
		return 0, nil
	}

	currentInodeNum := int32(0) // Comenzar desde el directorio raíz

	// Para cada componente del path, navegar por el sistema de archivos
	for i, component := range filteredComponents {
		fmt.Printf("DEBUG FindFileInodeByPath: Procesando componente %d: '%s', inodo actual: %d\n", i, component, currentInodeNum) // Leer el inodo actual
		inodeBytes := make([]byte, structs.INODE_SIZE)
		inodeOffset := int64(inodePos + (currentInodeNum * structs.INODE_SIZE))

		// Debug para identificar offsets negativos
		if inodeOffset < 0 {
			return -1, fmt.Errorf("offset negativo calculado: inodePos=%d, currentInodeNum=%d, offset=%d", inodePos, currentInodeNum, inodeOffset)
		}

		_, err := file.ReadAt(inodeBytes, inodeOffset)
		if err != nil {
			return -1, fmt.Errorf("error al leer el inodo: %v", err)
		}

		var inode structs.Inode
		buffer := bytes.NewReader(inodeBytes)
		binary.Read(buffer, binary.LittleEndian, &inode)

		if i == len(filteredComponents)-1 {
			// El inodo actual debe ser un directorio para buscar el archivo
			if inode.I_type != 1 {
				return -1, fmt.Errorf("'%s' no es un directorio, no se puede buscar el archivo", strings.Join(filteredComponents[:i], "/"))
			}
			// Buscar el archivo en el directorio
			return findFileInDirectory(file, component, inode, blockPos)
		} else {
			// Para componentes intermedios, el inodo actual debe ser un directorio
			if inode.I_type != 1 {
				return -1, fmt.Errorf("'%s' no es un directorio (I_type=%d)", component, inode.I_type)
			}
			// Buscar el siguiente directorio
			nextInodeNum, err := findFileInDirectory(file, component, inode, blockPos)
			if err != nil {
				return -1, err
			}
			currentInodeNum = nextInodeNum
		}
	}

	return currentInodeNum, nil
}
