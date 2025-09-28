package structs

type Jornal struct {
	J_count   int32             // Lleva el conteo del journal que es
	J_content InformationAccion // Cambiado el nombre para seguir convenciones Go
}

type InformationAccion struct {
	I_operation int32    // Contiene la operacion que se realizo
	I_path      [32]byte // Contiene la ruta del archivo
	I_content   [64]byte // Contiene todo el contenido (si es un archivo)
	I_date      float64  // Contiene la fecha de la operacion
}
