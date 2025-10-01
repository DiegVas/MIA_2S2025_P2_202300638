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
