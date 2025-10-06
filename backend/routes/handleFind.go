package routes

import (
	"fmt"
	"main/Analyzer"
	"main/provider/files"
	"main/utils"
	"net/http"
)

func handleFind(w http.ResponseWriter, command string) {
	params, err := Analyzer.ParseFind(command)
	if err != nil {
		response := utils.CreateResponse(
			err.Error(),
			nil,
			"system-error",
		)

		utils.SendJSONResponse(w, response, http.StatusBadRequest)
		return
	}

	result, err := files.FindFiles(params)
	if err != nil {
		response := utils.CreateResponse(
			err.Error(),
			params,
			"system-error",
		)

		utils.SendJSONResponse(w, response, http.StatusInternalServerError)
		return
	}

	// Crear mensaje descriptivo
	message := fmt.Sprintf("Búsqueda completada en '%s'. Se encontraron %d coincidencias para el patrón '%s'",
		params.Path, len(result.Matches), params.Name)

	message += "\nResultados:\n"
	for _, match := range result.Matches {
		message += fmt.Sprintf("- %s\n", match.Name)
	}

	response := utils.CreateResponse(
		message,
		result,
		"success",
	)

	utils.SendJSONResponse(w, response, http.StatusOK)
}
