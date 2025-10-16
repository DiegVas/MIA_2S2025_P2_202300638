package Analyzer

import (
	"errors"
	"fmt"
	"main/structs"
	"regexp"
	"strings"
)

// ParseRecovery analiza el comando RECOVERY y extrae los parámetros
func ParseRecovery(command string) (*structs.RecoveryParams, error) {
	params := &structs.RecoveryParams{}

	commandLower := strings.ToLower(command)
	if !strings.HasPrefix(commandLower, "recovery") {
		return nil, errors.New("el comando debe comenzar con 'recovery'")
	}

	paramStr := ""
	if strings.HasPrefix(commandLower, "recovery ") {
		paramStr = command[9:] // Remover "recovery "
	} else {
		paramStr = command[8:] // Remover "recovery"
	}
	paramStr = strings.TrimSpace(paramStr)

	paramRegex := regexp.MustCompile(`-(\w+)=(?:"([^"]*)"|'([^']*)'|([^\s]+))`)
	matches := paramRegex.FindAllStringSubmatch(paramStr, -1)

	if len(matches) == 0 {
		return nil, errors.New("no se encontraron parámetros válidos")
	}

	idFound := false
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
			params.ID = paramValue
			idFound = true
		default:
			return nil, fmt.Errorf("parámetro desconocido: '%s'", paramName)
		}
	}

	if !idFound {
		return nil, errors.New("falta el parámetro obligatorio: 'id'")
	}

	return params, nil
}

// ParseLoss analiza el comando LOSS y extrae los parámetros
func ParseLoss(command string) (*structs.LossParams, error) {
	params := &structs.LossParams{}

	commandLower := strings.ToLower(command)
	if !strings.HasPrefix(commandLower, "loss") {
		return nil, errors.New("el comando debe comenzar con 'loss'")
	}

	paramStr := ""
	if strings.HasPrefix(commandLower, "loss ") {
		paramStr = command[5:] // Remover "loss "
	} else {
		paramStr = command[4:] // Remover "loss"
	}
	paramStr = strings.TrimSpace(paramStr)

	paramRegex := regexp.MustCompile(`-(\w+)=(?:"([^"]*)"|'([^']*)'|([^\s]+))`)
	matches := paramRegex.FindAllStringSubmatch(paramStr, -1)

	if len(matches) == 0 {
		return nil, errors.New("no se encontraron parámetros válidos")
	}

	idFound := false
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
			params.ID = paramValue
			idFound = true
		default:
			return nil, fmt.Errorf("parámetro desconocido: '%s'", paramName)
		}
	}

	if !idFound {
		return nil, errors.New("falta el parámetro obligatorio: 'id'")
	}

	return params, nil
}

// ParseJournaling analiza el comando JOURNALING y extrae los parámetros
func ParseJournaling(command string) (*structs.JournalingParams, error) {
	params := &structs.JournalingParams{}

	commandLower := strings.ToLower(command)
	if !strings.HasPrefix(commandLower, "journaling") {
		return nil, errors.New("el comando debe comenzar con 'journaling'")
	}

	paramStr := ""
	if strings.HasPrefix(commandLower, "journaling ") {
		paramStr = command[11:] // Remover "journaling "
	} else {
		paramStr = command[10:] // Remover "journaling"
	}
	paramStr = strings.TrimSpace(paramStr)

	paramRegex := regexp.MustCompile(`-(\w+)=(?:"([^"]*)"|'([^']*)'|([^\s]+))`)
	matches := paramRegex.FindAllStringSubmatch(paramStr, -1)

	if len(matches) == 0 {
		return nil, errors.New("no se encontraron parámetros válidos")
	}

	idFound := false
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
			params.ID = paramValue
			idFound = true
		default:
			return nil, fmt.Errorf("parámetro desconocido: '%s'", paramName)
		}
	}

	if !idFound {
		return nil, errors.New("falta el parámetro obligatorio: 'id'")
	}

	return params, nil
}
