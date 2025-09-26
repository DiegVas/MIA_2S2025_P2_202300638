package routes

import (
	"fmt"
	"main/Analyzer"
	"main/provider/files"
	"main/provider/mounted"
	"main/structs"
	"main/utils"
	"net/http"
	"strings"
)

func handleCatFile(w http.ResponseWriter, command string) {
	params, err := Analyzer.ParseCatFile(command)
	if err != nil {
		response := utils.CreateResponse(
			err.Error(),
			nil,
			"system-error",
		)

		utils.SendJSONResponse(w, response, http.StatusBadRequest)
		return
	}

	mountManager := mounted.GetMountManager()

	paramsMount := &structs.MountParams{
		// Valor fijo para el carnet
		Carnet: "202300638",
		Name:   params.Name,
		Path:   params.Path,
	}

	id := ""

	if mountManager.IsMounted(params.Path, params.Name) {
		id, _ = mountManager.GetIDMountedPartition(params.Path, params.Name)
	} else {
		id_el, err := mounted.MountPartition(paramsMount)

		if err != nil {
			response := utils.CreateResponse(
				fmt.Sprintf("Error montando partición: %s", err.Error()),
				params,
				"system-error",
			)
			utils.SendJSONResponse(w, response, http.StatusBadRequest)
			return
		}

		id = id_el

	}

	content, err := files.ReadFileContent(params.Files, id)

	if err != nil {
		response := utils.CreateResponse(
			fmt.Sprintf("Error al leer '%s': %s", params.Files, err.Error()),
			params,
			"system-error",
		)

		utils.SendJSONResponse(w, response, http.StatusBadRequest)
		return
	}

	response := utils.CreateResponse(
		content,
		params,
		"system",
	)

	utils.SendJSONResponse(w, response, http.StatusOK)

}

func handleCat(w http.ResponseWriter, command string) {
	params, err := Analyzer.ParseCat(command)
	if err != nil {
		response := utils.CreateResponse(
			err.Error(),
			nil,
			"system-error",
		)

		utils.SendJSONResponse(w, response, http.StatusBadRequest)
		return
	}

	var results []string
	var hasErrors bool

	for _, filePath := range params.Files {
		content, err := files.ReadFileContent(filePath, "")
		if err != nil {
			results = append(results, fmt.Sprintf("Error al leer '%s': %s", filePath, err.Error()))
			hasErrors = true
		} else {
			results = append(results, content)
		}
	}

	finalContent := strings.Join(results, "\n")

	var messageType string
	if hasErrors {
		messageType = "system-error"
	} else {
		messageType = "system"
	}

	response := utils.CreateResponse(
		finalContent,
		params,
		messageType,
	)

	utils.SendJSONResponse(w, response, http.StatusOK)
}
