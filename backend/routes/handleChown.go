package routes

import (
	"fmt"
	"main/Analyzer"
	"main/provider/files"
	"main/utils"
	"net/http"
)

func handleChown(w http.ResponseWriter, command string) {
	params, err := Analyzer.ParseChown(command)
	if err != nil {
		response := utils.CreateResponse(
			err.Error(),
			nil,
			"system-error",
		)

		utils.SendJSONResponse(w, response, http.StatusBadRequest)
		return
	}

	err = files.ChangeOwner(params)
	if err != nil {
		response := utils.CreateResponse(
			err.Error(),
			params,
			"system-error",
		)

		utils.SendJSONResponse(w, response, http.StatusInternalServerError)
		return
	}

	var message string
	if params.R {
		message = fmt.Sprintf("Propietario cambiado recursivamente a '%s' para '%s'", params.Usuario, params.Path)
	} else {
		message = fmt.Sprintf("Propietario cambiado a '%s' para '%s'", params.Usuario, params.Path)
	}

	response := utils.CreateResponse(
		message,
		params,
		"system",
	)

	utils.SendJSONResponse(w, response, http.StatusOK)
}
