package SystemFiles

import (
	"fmt"
	"strings"

	"main/provider/mounted"
	"main/provider/reports"

	"main/structs"
	"os"
)

// DirectoryListing representa el resultado del listado
type DirectoryListing struct {
	Path      string          `json:"path"`
	Entries   []DirectoryData `json:"entries"`
	IsMounted bool            `json:"is_mounted"`
}

type DirectoryData struct {
	Date     string `json:"date"`
	Size     string `json:"size"`
	FileType string `json:"file_type"`
	Name     string `json:"name"`
}

// ListDirectory lista el contenido de un directorio en una partición montada
func ListDirectory(params *structs.LSParams) (*DirectoryListing, error) {
	fmt.Printf("DEBUG - ListDirectory iniciado con ID: '%s', Of: '%s'\n", params.Name, params.Of)

	// Validar parámetros
	if params.Name == "" {
		return nil, fmt.Errorf("el nombre de la partición es obligatorio")
	}

	mountManager := mounted.GetMountManager()

	paramsMount := &structs.MountParams{
		// Valor fijo para el carnet
		Carnet: "202300638",
		Name:   params.Name,
		Path:   params.Path,
	}
	id := ""

	if mountManager.IsMounted(params.Path, params.Name) {
		id, _ = mountManager.GetIDMountedPartition(params.Path, params.Name)

		fmt.Printf("DEBUG - Partición ya montada con ID: '%s'\n", id)
	} else {
		id_el, err := mounted.MountPartition(paramsMount)
		id = id_el
		if err != nil {
			fmt.Printf("DEBUG - ERROR montando partición: %v\n", err)
			return nil, fmt.Errorf("error montando partición: %v", err)
		}
	}

	mountedPartition, found := mountManager.GetMountedPartition(id)

	if !found {
		fmt.Printf("DEBUG - ERROR: Partición con ID '%s' no está montada\n", id)

		return nil, fmt.Errorf("la partición con ID '%s' no está montada. Debe montar la partición primero", id)
	}
	fmt.Printf("DEBUG - Partición encontrada: Path='%s', Name='%s'\n", mountedPartition.Path, mountedPartition.Name)

	// Leer el superbloque
	superblock, err := reports.ReadSuperblock(mountedPartition.Path, id)
	if err != nil {
		return nil, fmt.Errorf("error leyendo superbloque: %v", err)
	}

	// Abrir el archivo del disco
	file, err := os.Open(mountedPartition.Path)
	if err != nil {
		return nil, fmt.Errorf("error abriendo disco: %v", err)
	}
	defer file.Close()

	//Buscar el inodo del directorio especificado
	dirInodeNum, err := reports.FindInodeByPath(file, params.Of, superblock)
	if err != nil {
		return nil, fmt.Errorf("error encontrando directorio '%s': %v", params.Of, err)
	}

	// Leer el inodo del directorio
	dirInode, err := reports.ReadInodeAt(file, dirInodeNum, superblock.S_inode_start)
	if err != nil {
		return nil, fmt.Errorf("error leyendo inodo del directorio: %v", err)
	}

	// Verificar que sea un directorio
	if dirInode.I_type != 1 {
		return nil, fmt.Errorf("'%s' no es un directorio", params.Of)
	}

	// Obtener entradas del directorio
	entries, err := reports.GetDirectoryEntries(file, dirInode, superblock)
	if err != nil {
		return nil, fmt.Errorf("error leyendo entradas del directorio: %v", err)
	}

	fmt.Printf("DEBUG - Entradas encontradas en '%s': %d\n", params.Of, len(entries))

	directoryResult := []DirectoryData{}

	for _, entry := range entries {
		if entry.Name == "" {
			continue
		}

		entryInode, err := reports.ReadInodeAt(file, entry.InodeNum, superblock.S_inode_start)
		if err != nil {
			fmt.Printf("DEBUG - ERROR leyendo inodo de entrada '%s': %v\n", entry.Name, err)
			continue
		}

		size := fmt.Sprintf("%d", entryInode.I_size)

		date := strings.TrimRight(string(entryInode.I_ctime[:]), "\x00")

		fileType := "file"
		if entryInode.I_type == 1 {
			fileType = "directory"
		}

		directoryResult = append(directoryResult, DirectoryData{
			Date:     date,
			Size:     size,
			FileType: fileType,
			Name:     entry.Name,
		})

	}

	// Retornar el listado

	fmt.Printf("DEBUG - ListDirectory completado. Total entradas: %d\n", len(directoryResult))

	fmt.Printf("DEBUG - ListDirectory contenido: %+v\n", directoryResult)

	return &DirectoryListing{
		Path:    params.Of,
		Entries: directoryResult,
	}, nil
}
