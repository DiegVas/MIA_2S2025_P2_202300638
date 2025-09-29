package Analyzer

import (
	"errors"
	"fmt"
	"main/structs"
	"regexp"
	"strings"
)

func ParseUnmount(command string) (*structs.UnmountParams, error) {
	params := &structs.UnmountParams{
		ID: "",
	}

	originalCommand := command
	commandLower := strings.ToLower(command)

	if !strings.HasPrefix(commandLower, "unmount") {
		return nil, errors.New("el comando debe comenzar con 'unmount'")
	}

	paramStr := ""
	if strings.HasPrefix(commandLower, "unmount ") {
		paramStr = originalCommand[8:]
	} else {
		paramStr = originalCommand[7:]
	}
	paramStr = strings.TrimSpace(paramStr)

	paramRegex := regexp.MustCompile(`-(\w+)=(?:"([^"]*)"|'([^']*)'|([^\s]+))`)
	matches := paramRegex.FindAllStringSubmatch(paramStr, -1)

	if len(matches) == 0 {
		return nil, errors.New("no se encontraron parámetros válidos")
	}

	requiredParams := map[string]bool{
		"id": false,
	}

	for _, match := range matches {
		if len(match) < 3 {
			continue
		}

		paramName := strings.ToLower(match[1])

		var paramValue string
		if match[2] != "" {
			paramValue = match[2]
		} else if match[3] != "" {
			paramValue = match[3]
		} else if match[4] != "" {
			paramValue = match[4]
		}

		switch paramName {
		case "id":
			if paramValue == "" {
				return nil, errors.New("el parámetro id no puede estar vacío")
			}
			params.ID = paramValue
			requiredParams["id"] = true

		default:
			return nil, fmt.Errorf("parámetro desconocido: '%s'", paramName)
		}
	}

	for param, found := range requiredParams {
		if !found {
			return nil, fmt.Errorf("falta el parámetro obligatorio: '%s'", param)
		}
	}

	return params, nil
}
