package Analyzer

import (
	"fmt"
	"main/structs"
	"regexp"
	"strings"
)

// ParseLS parsea el comando LS
func ParseLS(command string) (*structs.LSParams, error) {
	params := &structs.LSParams{
		Path: "",
		Name: "",
		Of:   "/", // Valor por defecto es la raíz
	}

	// Expresión regular para capturar los parámetros
	re := regexp.MustCompile(`-(\w+)=([^\s]+)`)
	matches := re.FindAllStringSubmatch(command, -1)

	for _, match := range matches {
		param := strings.ToLower(match[1])
		value := strings.Trim(match[2], "\"")

		switch param {
		case "path":
			params.Path = value
		case "name":
			params.Name = value
		case "of":
			params.Of = value
		default:
			return nil, fmt.Errorf("parámetro no reconocido: %s", param)
		}
	}

	// Validaciones
	if params.Path == "" {
		return nil, fmt.Errorf("el parámetro -path es requerido")
	}

	if params.Name == "" {
		return nil, fmt.Errorf("el parámetro -name es requerido")
	}

	return params, nil
}
