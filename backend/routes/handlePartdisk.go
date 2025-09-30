package routes

import (
	"fmt"
	"main/Analyzer"
	diskmanager "main/diskManager"
	"main/structs"
	"main/utils"
	"net/http"
)

func handlePartdisk(w http.ResponseWriter, command string) {
	params, err := Analyzer.ParsePartdisk(command)

	if err != nil {
		fmt.Printf("handlePartdisk - error: %s\n", err.Error())

		response := structs.Response{
			Message: fmt.Sprintf("Error en el comando: %s", err.Error()),
			Data:    nil,
			Type:    "system-error",
		}

		utils.SendJSONResponse(w, response, http.StatusBadRequest)
		return
	}

	// Obtener información de las particiones del disco
	partitions, err := diskmanager.GetDiskPartitions(params.Path)

	if err != nil {
		response := structs.Response{
			Message: fmt.Sprintf("Error al obtener particiones del disco: %s", err.Error()),
			Data:    nil,
			Type:    "system-error",
		}

		utils.SendJSONResponse(w, response, http.StatusInternalServerError)
		return
	}

	confirmationMessage := fmt.Sprintf("Información de particiones obtenida exitosamente para el disco: %s", params.Path)

	response := structs.Response{
		Message: confirmationMessage,
		Data:    partitions,
		Type:    "system",
	}

	utils.SendJSONResponse(w, response, http.StatusOK)
}
