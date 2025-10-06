package Analyzer

import (
	"errors"
	"main/structs"
	"regexp"
	"strings"
)

func ParseMove(command string, command_parse string) (*structs.MoveParams, error) {
	var params structs.MoveParams

	// Verificar que el comando comience con move
	if !strings.HasPrefix(strings.ToLower(strings.TrimSpace(command)), command_parse) {
		return nil, errors.New("comando inválido, debe comenzar con 'move'")
	}

	// Regex para capturar parámetros con o sin comillas
	pathRegex := regexp.MustCompile(`-path=(?:"([^"]+)"|([^\s]+))`)
	destinoRegex := regexp.MustCompile(`-destino=(?:"([^"]+)"|([^\s]+))`)

	// Buscar path
	pathMatch := pathRegex.FindStringSubmatch(command)
	if pathMatch == nil {
		return nil, errors.New("el parámetro -path es obligatorio")
	}

	if pathMatch[1] != "" {
		params.Path = pathMatch[1] // Path con comillas
	} else {
		params.Path = pathMatch[2] // Path sin comillas
	}

	// Buscar destino
	destinoMatch := destinoRegex.FindStringSubmatch(command)
	if destinoMatch == nil {
		return nil, errors.New("el parámetro -destino es obligatorio")
	}

	if destinoMatch[1] != "" {
		params.Destino = destinoMatch[1] // Destino con comillas
	} else {
		params.Destino = destinoMatch[2] // Destino sin comillas
	}

	// Validaciones
	if params.Path == "" {
		return nil, errors.New("el parámetro -path no puede estar vacío")
	}

	if params.Destino == "" {
		return nil, errors.New("el parámetro -destino no puede estar vacío")
	}

	// Validar que las rutas comiencen con /
	if !strings.HasPrefix(params.Path, "/") {
		return nil, errors.New("la ruta de origen debe ser absoluta (comenzar con /)")
	}

	if !strings.HasPrefix(params.Destino, "/") {
		return nil, errors.New("la ruta de destino debe ser absoluta (comenzar con /)")
	}

	// Validar que no se trate de mover a sí mismo
	if params.Path == params.Destino {
		return nil, errors.New("la ruta de origen y destino no pueden ser la misma")
	}

	return &params, nil
}
