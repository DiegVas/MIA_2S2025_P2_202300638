package Analyzer

import (
	"errors"
	"main/structs"
	"regexp"
)

func ParseRename(command string) (*structs.RenameFileParams, error) {
	params := &structs.RenameFileParams{}

	pathRegex := regexp.MustCompile(`-path=(?:"([^"]+)"|([^\s]+))`)
	nameRegex := regexp.MustCompile(`-name=(?:"([^"]+)"|([^\s]+))`)

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

	// Extraer el parámetro -name
	nameMatch := nameRegex.FindStringSubmatch(command)
	if nameMatch == nil {
		return nil, errors.New("el parámetro -name es obligatorio")
	}

	if nameMatch[1] != "" {
		params.NewName = nameMatch[1]
	} else {
		params.NewName = nameMatch[2]
	}

	if params.NewName == "" {
		return nil, errors.New("el parámetro -name no puede estar vacío")
	}

	return params, nil
}
