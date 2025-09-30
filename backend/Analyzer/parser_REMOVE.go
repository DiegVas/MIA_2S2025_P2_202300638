package Analyzer

import (
	"errors"
	"main/structs"
	"regexp"
	"strings"
)

func ParseRemove(command string) (*structs.RemoveParams, error) {
	var params structs.RemoveParams

	// Verificar que el comando comience con remove
	if !strings.HasPrefix(strings.ToLower(strings.TrimSpace(command)), "remove") {
		return nil, errors.New("comando inválido, debe comenzar con 'remove'")
	}

	// Regex para path (con o sin comillas)
	pathRegex := regexp.MustCompile(`-path=(?:"([^"]+)"|([^\s]+))`)

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

	// Validaciones
	if params.Path == "" {
		return nil, errors.New("el parámetro -path no puede estar vacío")
	}

	// Validar que la ruta comience con /
	if !strings.HasPrefix(params.Path, "/") {
		return nil, errors.New("la ruta debe ser absoluta (comenzar con /)")
	}

	return &params, nil
}
