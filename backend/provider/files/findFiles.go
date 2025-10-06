package files

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"main/provider/common"
	"main/provider/mounted"
	"main/provider/reports"
	"main/structs"
	"os"
)

// FindResult representa el resultado de una búsqueda
type FindResult struct {
	Path      string                  `json:"path"`
	Matches   []structs.DirectoryData `json:"matches"`
	IsMounted bool                    `json:"is_mounted"`
}

// FindFiles busca archivos y carpetas que coincidan con el patrón especificado
func FindFiles(params *structs.FindParams) (*FindResult, error) {
	// Verificar que haya una sesión activa
	session := structs.GetSession()
	if !session.Active {
		return nil, fmt.Errorf("debe iniciar sesión para usar el comando FIND")
	}

	// Obtener la partición montada
	mountManager := mounted.GetMountManager()
	partition, found := mountManager.GetMountedPartition(session.ID)
	if !found {
		return nil, fmt.Errorf("no hay ninguna partición montada para esta sesión")
	}

	// Leer el superbloque
	superblock, err := common.ReadSuperblock(partition.Path, session.ID)
	if err != nil {
		return nil, fmt.Errorf("error leyendo superbloque: %v", err)
	}

	// Abrir el archivo del disco
	file, err := os.Open(partition.Path)
	if err != nil {
		return nil, fmt.Errorf("error abriendo disco: %v", err)
	}
	defer file.Close()

	// Buscar el inodo del directorio especificado
	dirInodeNum, err := reports.FindInodeByPath(file, params.Path, superblock)
	if err != nil {
		return nil, fmt.Errorf("error encontrando directorio '%s': %v", params.Path, err)
	}

	// Leer el inodo del directorio
	dirInode, err := reports.ReadInodeAt(file, dirInodeNum, superblock.S_inode_start)
	if err != nil {
		return nil, fmt.Errorf("error leyendo inodo del directorio: %v", err)
	}

	// Verificar que sea un directorio
	if dirInode.I_type != 1 {
		return nil, fmt.Errorf("'%s' no es un directorio", params.Path)
	}

	// Inicializar resultado
	result := &FindResult{
		Path:      params.Path,
		Matches:   []structs.DirectoryData{},
		IsMounted: true,
	}

	// Realizar búsqueda recursiva
	err = searchRecursive(file, params.Path, dirInodeNum, params.Name, superblock, result)
	if err != nil {
		return nil, fmt.Errorf("error durante la búsqueda: %v", err)
	}

	return result, nil
}

// searchRecursive realiza la búsqueda recursiva en el sistema de archivos
func searchRecursive(file *os.File, currentPath string, dirInodeNum int32, pattern string,
	superblock *structs.Superblock, result *FindResult) error {

	// Leer el inodo del directorio actual
	dirInode, err := reports.ReadInodeAt(file, dirInodeNum, superblock.S_inode_start)
	if err != nil {
		return fmt.Errorf("error leyendo inodo del directorio: %v", err)
	}

	// Obtener entradas del directorio
	entries, err := reports.GetDirectoryEntries(file, dirInode, superblock)
	if err != nil {
		return fmt.Errorf("error leyendo entradas del directorio: %v", err)
	}

	// Procesar cada entrada
	for _, entry := range entries {
		if entry.Name == "" || entry.Name == "." || entry.Name == ".." {
			continue
		}

		// Construir la ruta completa
		fullPath := filepath.Join(currentPath, entry.Name)
		if currentPath == "/" {
			fullPath = "/" + entry.Name
		}

		// Leer el inodo de la entrada
		entryInode, err := reports.ReadInodeAt(file, entry.InodeNum, superblock.S_inode_start)
		if err != nil {
			continue // Continuar con la siguiente entrada si hay error
		}

		// Verificar si coincide con el patrón
		if matchesPattern(entry.Name, pattern) {
			// Crear datos del directorio
			size := fmt.Sprintf("%d", entryInode.I_size)
			date := strings.TrimRight(string(entryInode.I_ctime[:]), "\x00")

			fileType := "file"
			if entryInode.I_type == 1 {
				fileType = "directory"
			}

			// Agregar al resultado
			result.Matches = append(result.Matches, structs.DirectoryData{
				Date:     date,
				Size:     size,
				FileType: fileType,
				Name:     fullPath, // Usar la ruta completa como nombre
			})
		}

		// Si es un directorio, buscar recursivamente
		if entryInode.I_type == 1 {
			err = searchRecursive(file, fullPath, entry.InodeNum, pattern, superblock, result)
			if err != nil {
				// Continuar con otros directorios aunque uno falle
				fmt.Printf("Warning: Error searching in directory %s: %v\n", fullPath, err)
			}
		}
	}

	return nil
}

// matchesPattern verifica si un nombre coincide con el patrón usando * y ?
func matchesPattern(name, pattern string) bool {
	// Si el patrón es exactamente "*", coincide con todo
	if pattern == "*" {
		return true
	}

	// Convertir el patrón a regex
	// ? -> . (un solo carácter)
	// * -> .* (cero o más caracteres)
	regexPattern := strings.ReplaceAll(pattern, "?", ".")
	regexPattern = strings.ReplaceAll(regexPattern, "*", ".*")

	// Agregar anclas de inicio y fin para coincidencia exacta
	regexPattern = "^" + regexPattern + "$"

	// Compilar y probar el regex
	regex, err := regexp.Compile(regexPattern)
	if err != nil {
		// Si el regex no es válido, hacer coincidencia exacta
		return name == pattern
	}

	return regex.MatchString(name)
}
