package routes

import (
	"fmt"
	"main/Analyzer"
	SystemFiles "main/provider/systemFiles"
	"main/structs"
	"main/utils"
	"net/http"
)

func handleLS(w http.ResponseWriter, command string) {
	params, err := Analyzer.ParseLS(command)
	if err != nil {
		fmt.Printf("handleLS - error: %s\n", err.Error())

		response := structs.Response{
			Message: fmt.Sprintf("Error en el comando: %s", err.Error()),
			Data:    nil,
			Type:    "system-error",
		}

		utils.SendJSONResponse(w, response, http.StatusBadRequest)
		return
	}

	// Obtener el listado de archivos y carpetas
	listing, err := SystemFiles.ListDirectory(params)
	if err != nil {
		response := structs.Response{
			Message: fmt.Sprintf("Error al listar directorio: %s", err.Error()),
			Data:    nil,
			Type:    "system-error",
		}

		utils.SendJSONResponse(w, response, http.StatusInternalServerError)
		return
	}

	confirmationMessage := fmt.Sprintf("Listado obtenido exitosamente - Directorio: %s en partición %s", params.Of, params.Name)

	response := structs.Response{
		Message: confirmationMessage,
		Data:    listing,
		Type:    "system",
	}

	utils.SendJSONResponse(w, response, http.StatusOK)
}
