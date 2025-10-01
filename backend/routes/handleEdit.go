package routes

import (
	"fmt"
	"main/Analyzer"
	"main/provider/files"
	"main/utils"
	"net/http"
)

func handleEdit(w http.ResponseWriter, command string) {
	params, err := Analyzer.ParseEdit(command)
	if err != nil {
		response := utils.CreateResponse(
			err.Error(),
			nil,
			"system-error",
		)

		utils.SendJSONResponse(w, response, http.StatusBadRequest)
		return
	}

	err = files.EditFile(params)
	if err != nil {
		response := utils.CreateResponse(
			err.Error(),
			params,
			"system-error",
		)

		utils.SendJSONResponse(w, response, http.StatusInternalServerError)
		return
	}

	message := fmt.Sprintf("Archivo '%s' editado exitosamente con contenido desde '%s'", params.Path, params.Contenido)

	response := utils.CreateResponse(
		message,
		params,
		"system",
	)

	utils.SendJSONResponse(w, response, http.StatusOK)
}
