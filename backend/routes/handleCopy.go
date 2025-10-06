package routes

import (
	"fmt"
	"main/Analyzer"
	"main/provider/files"
	"main/utils"
	"net/http"
)

func handleCopy(w http.ResponseWriter, command string) {
	params, err := Analyzer.ParseMove(command, "copy")
	if err != nil {
		response := utils.CreateResponse(
			err.Error(),
			nil,
			"system-error",
		)

		utils.SendJSONResponse(w, response, http.StatusBadRequest)
		return
	}

	err = files.CopyFileOrDirectory(params)
	if err != nil {
		response := utils.CreateResponse(
			err.Error(),
			params,
			"system-error",
		)

		utils.SendJSONResponse(w, response, http.StatusInternalServerError)
		return
	}

	message := fmt.Sprintf("Archivo o directorio movido exitosamente de '%s' a '%s'", params.Path, params.Destino)

	response := utils.CreateResponse(
		message,
		params,
		"success",
	)

	utils.SendJSONResponse(w, response, http.StatusOK)
}
