package Analyzer

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

type CatParams struct {
	Files []string
}

type CatFileParams struct {
	Files string
	Path  string
	Name  string
}

func ParseCatFile(command string) (*CatFileParams, error) {
	params := &CatFileParams{
		Files: "",
		Path:  "",
		Name:  "",
	}

	// Regex corregida para capturar parámetro=valor
	re := regexp.MustCompile(`-(\w+)=(?:"([^"]+)"|([^\s]+))`)
	matches := re.FindAllStringSubmatch(command, -1)

	for _, match := range matches {
		param := strings.ToLower(match[1]) // nombre del parámetro
		var value string
		if match[2] != "" {
			value = match[2] // valor con comillas
		} else {
			value = match[3] // valor sin comillas
		}

		switch param {
		case "path":
			params.Path = value
		case "name":
			params.Name = value
		case "file":
			params.Files = value
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

	if params.Files == "" {
		return nil, fmt.Errorf("el parámetro -file es requerido")
	}

	return params, nil
}

func ParseCat(command string) (*CatParams, error) {

	re := regexp.MustCompile(`-file(\d+)=(?:"([^"]+)"|([^\s]+))`)
	matches := re.FindAllStringSubmatch(command, -1)

	if len(matches) == 0 {
		return nil, errors.New("el comando CAT requiere al menos un parámetro -file1")
	}

	params := &CatParams{
		Files: make([]string, 0),
	}

	fileMap := make(map[string]string)
	maxFileNum := 0

	for _, match := range matches {
		fileNum := match[1]
		var filePath string

		if match[2] != "" {
			filePath = match[2]
		} else {
			filePath = match[3]
		}

		if filePath == "" {
			return nil, errors.New("ruta de archivo vacía en parámetro -file" + fileNum)
		}

		fileMap[fileNum] = filePath

		num := 0
		for i, char := range fileNum {
			if i == 0 && len(fileNum) > 1 {

				num = int(char-'0') * 10
			} else {
				num += int(char - '0')
			}
		}
		if num > maxFileNum {
			maxFileNum = num
		}
	}

	for i := 1; i <= maxFileNum; i++ {
		fileNumStr := string(rune(i + '0'))
		if i >= 10 {

			fileNumStr = string(rune((i/10)+'0')) + string(rune((i%10)+'0'))
		}

		if filePath, exists := fileMap[fileNumStr]; exists {
			params.Files = append(params.Files, filePath)
		}
	}

	if len(params.Files) == 0 {
		return nil, errors.New("no se encontraron archivos válidos en los parámetros")
	}

	return params, nil
}
