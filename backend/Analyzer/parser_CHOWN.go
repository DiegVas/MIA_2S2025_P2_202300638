package Analyzer

import (
	"errors"
	"main/structs"
	"regexp"
	"strings"
)

// ParseChown parsea el comando chown y retorna los parámetros
func ParseChown(command string) (*structs.ChownParams, error) {
	// Expresión regular para extraer los parámetros
	re := regexp.MustCompile(`chown\s+(.*)`)
	matches := re.FindStringSubmatch(command)

	if len(matches) < 2 {
		return nil, errors.New("formato de comando chown inválido")
	}

	// Extraer la parte de parámetros
	paramString := strings.TrimSpace(matches[1])

	params := &structs.ChownParams{}

	// Dividir por espacios, pero manteniendo las comillas
	parts := splitRespectingQuotes(paramString)

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		if strings.HasPrefix(part, "-path=") {
			path := strings.TrimPrefix(part, "-path=")
			path = strings.Trim(path, "\"")
			params.Path = path
		} else if strings.HasPrefix(part, "-usuario=") {
			usuario := strings.TrimPrefix(part, "-usuario=")
			usuario = strings.Trim(usuario, "\"")
			params.Usuario = usuario
		} else if part == "-r" {
			params.R = true
		}
	}

	// Validaciones
	if params.Path == "" {
		return nil, errors.New("el parámetro -path es obligatorio")
	}

	if params.Usuario == "" {
		return nil, errors.New("el parámetro -usuario es obligatorio")
	}

	return params, nil
}

// splitRespectingQuotes divide una cadena por espacios pero respeta las comillas
func splitRespectingQuotes(s string) []string {
	var result []string
	var current strings.Builder
	inQuotes := false

	for _, char := range s {
		if char == '"' {
			inQuotes = !inQuotes
			current.WriteRune(char)
		} else if char == ' ' && !inQuotes {
			if current.Len() > 0 {
				result = append(result, current.String())
				current.Reset()
			}
		} else {
			current.WriteRune(char)
		}
	}

	if current.Len() > 0 {
		result = append(result, current.String())
	}

	return result
}
