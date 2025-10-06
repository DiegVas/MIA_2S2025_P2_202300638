package routes

import (
	"fmt"
	"main/Analyzer"
	"main/provider/files"
	"main/utils"
	"net/http"
)

func handleChmod(w http.ResponseWriter, command string) {
	params, err := Analyzer.ParseChmod(command)
	if err != nil {
		response := utils.CreateResponse(
			err.Error(),
			nil,
			"system-error",
		)

		utils.SendJSONResponse(w, response, http.StatusBadRequest)
		return
	}

	err = files.ChangePermissions(params)
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
		message = fmt.Sprintf("Permisos cambiados recursivamente a '%s' para '%s'", params.UGO, params.Path)
	} else {
		message = fmt.Sprintf("Permisos cambiados a '%s' para '%s'", params.UGO, params.Path)
	}

	response := utils.CreateResponse(
		message,
		params,
		"system",
	)

	utils.SendJSONResponse(w, response, http.StatusOK)
}
