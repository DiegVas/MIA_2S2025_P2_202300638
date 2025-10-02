package routes

import (
	"fmt"
	"main/Analyzer"
	"main/provider/files"
	"main/utils"
	"net/http"
)

func handleRename(w http.ResponseWriter, command string) {
	params, err := Analyzer.ParseRename(command)
	if err != nil {
		response := utils.CreateResponse(
			err.Error(),
			nil,
			"system-error",
		)

		utils.SendJSONResponse(w, response, http.StatusBadRequest)
		return
	}

	err = files.RenameFile(params)
	if err != nil {
		response := utils.CreateResponse(
			err.Error(),
			params,
			"system-error",
		)

		utils.SendJSONResponse(w, response, http.StatusInternalServerError)
		return
	}

	message := fmt.Sprintf("Archivo/directorio renombrado exitosamente a '%s'", params.NewName)

	response := utils.CreateResponse(
		message,
		params,
		"system",
	)

	utils.SendJSONResponse(w, response, http.StatusOK)
}
