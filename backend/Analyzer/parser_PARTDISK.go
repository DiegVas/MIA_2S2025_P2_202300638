package Analyzer

import (
	"fmt"
	"main/structs"
	"regexp"
)

func ParsePartdisk(command string) (*structs.PartdiskParams, error) {
	// Definir regex para capturar el parámetro -path
	pathRegex := regexp.MustCompile(`-path\s*=\s*([^\s]+)`)

	// Buscar el path
	pathMatch := pathRegex.FindStringSubmatch(command)
	if pathMatch == nil {
		return nil, fmt.Errorf("parámetro -path es obligatorio")
	}

	path := pathMatch[1]

	// Crear y retornar los parámetros
	params := &structs.PartdiskParams{
		Path: path,
	}

	return params, nil
}
