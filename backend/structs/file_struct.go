package structs

type MkfileParams struct {
	Path    string
	R       bool
	Size    int32
	Cont    string
	SizeSet bool
	ContSet bool
}

type EditParams struct {
	Path      string // Ruta del archivo a editar
	Contenido string // Ruta del archivo local con el nuevo contenido
}

type RenameFileParams struct {
	Path    string // Ruta del archivo a renombrar
	NewName string // Nuevo nombre del archivo
}

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
