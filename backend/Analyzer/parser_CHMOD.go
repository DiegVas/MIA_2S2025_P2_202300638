package Analyzer

import (
	"errors"
	"main/structs"
	"regexp"
	"strconv"
	"strings"
)

// ParseChmod parsea el comando chmod y retorna los parámetros
func ParseChmod(command string) (*structs.ChmodParams, error) {
	// Expresión regular para extraer los parámetros
	re := regexp.MustCompile(`chmod\s+(.*)`)
	matches := re.FindStringSubmatch(command)

	if len(matches) < 2 {
		return nil, errors.New("formato de comando chmod inválido")
	}

	// Extraer la parte de parámetros
	paramString := strings.TrimSpace(matches[1])

	params := &structs.ChmodParams{}

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
		} else if strings.HasPrefix(part, "-ugo=") {
			ugo := strings.TrimPrefix(part, "-ugo=")
			ugo = strings.Trim(ugo, "\"")
			params.UGO = ugo
		} else if part == "-r" {
			params.R = true
		}
	}

	// Validaciones
	if params.Path == "" {
		return nil, errors.New("el parámetro -path es obligatorio")
	}

	if params.UGO == "" {
		return nil, errors.New("el parámetro -ugo es obligatorio")
	}

	// Validar formato UGO
	if err := validateUGOFormat(params.UGO); err != nil {
		return nil, err
	}

	return params, nil
}

// validateUGOFormat valida que el formato UGO sea correcto (3 dígitos del 0-7)
func validateUGOFormat(ugo string) error {
	// Debe tener exactamente 3 caracteres
	if len(ugo) != 3 {
		return errors.New("el parámetro -ugo debe tener exactamente 3 dígitos")
	}

	// Cada dígito debe estar entre 0 y 7
	for _, char := range ugo {
		digit, err := strconv.Atoi(string(char))
		if err != nil {
			return errors.New("el parámetro -ugo debe contener solo números")
		}

		if digit < 0 || digit > 7 {
			return errors.New("cada dígito en -ugo debe estar entre 0 y 7")
		}
	}

	return nil
}
