package Analyzer

import (
	"errors"
	"main/structs"
	"regexp"
	"strings"
)

func ParseFind(command string) (*structs.FindParams, error) {
	var params structs.FindParams

	// Verificar que el comando comience con find
	if !strings.HasPrefix(strings.ToLower(strings.TrimSpace(command)), "find") {
		return nil, errors.New("comando inválido, debe comenzar con 'find'")
	}

	// Regex para capturar parámetros con o sin comillas, permitiendo espacios alrededor del =
	pathRegex := regexp.MustCompile(`-path\s*=\s*(?:"([^"]+)"|([^\s]+))`)
	nameRegex := regexp.MustCompile(`-name\s*=\s*(?:"([^"]+)"|([^\s]+))`)

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

	// Buscar name
	nameMatch := nameRegex.FindStringSubmatch(command)
	if nameMatch == nil {
		return nil, errors.New("el parámetro -name es obligatorio")
	}

	if nameMatch[1] != "" {
		params.Name = nameMatch[1] // Name con comillas
	} else {
		params.Name = nameMatch[2] // Name sin comillas
	}

	// Validaciones
	if params.Path == "" {
		return nil, errors.New("el parámetro -path no puede estar vacío")
	}

	if params.Name == "" {
		return nil, errors.New("el parámetro -name no puede estar vacío")
	}

	// Validar que la ruta comience con /
	if !strings.HasPrefix(params.Path, "/") {
		return nil, errors.New("la ruta debe ser absoluta (comenzar con /)")
	}

	return &params, nil
}
