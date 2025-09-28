package structs

// MkfsParams estructura para los parámetros del comando MKFS
type MkfsParams struct {
	ID   string // ID de la partición montada
	Type string // Tipo de formateo (full por defecto)
	FS   string // Sistema de archivos (2fs para EXT2, 3fs para EXT3) - por defecto EXT2
}

func (p *MkfsParams) String() string {
	return "ID: " + p.ID + ", Type: " + p.Type + ", FS: " + p.FS
}
