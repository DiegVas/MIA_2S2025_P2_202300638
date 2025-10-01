package Analyzer

import (
	"errors"
	"main/structs"
	"regexp"
)

func ParseEdit(command string) (*structs.EditParams, error) {
	params := &structs.EditParams{}

	pathRegex := regexp.MustCompile(`-path=(?:"([^"]+)"|([^\s]+))`)
	contenidoRegex := regexp.MustCompile(`-contenido=(?:"([^"]+)"|([^\s]+))`)

	// Extraer el parámetro -path
	pathMatch := pathRegex.FindStringSubmatch(command)
	if pathMatch == nil {
		return nil, errors.New("el parámetro -path es obligatorio")
	}

	if pathMatch[1] != "" {
		params.Path = pathMatch[1]
	} else {
		params.Path = pathMatch[2]
	}

	if params.Path == "" {
		return nil, errors.New("el parámetro -path no puede estar vacío")
	}

	// Extraer el parámetro -contenido
	contenidoMatch := contenidoRegex.FindStringSubmatch(command)
	if contenidoMatch == nil {
		return nil, errors.New("el parámetro -contenido es obligatorio")
	}

	if contenidoMatch[1] != "" {
		params.Contenido = contenidoMatch[1]
	} else {
		params.Contenido = contenidoMatch[2]
	}

	if params.Contenido == "" {
		return nil, errors.New("el parámetro -contenido no puede estar vacío")
	}

	return params, nil
}
