package routes

import (
	"fmt"
	"main/Analyzer"
	"main/provider/mounted"
	"main/structs"
	"main/utils"
	"net/http"
)

func handleUnmount(w http.ResponseWriter, command string) {
	params, err := Analyzer.ParseUnmount(command)

	if err != nil {
		fmt.Printf("handleUnmount - error: %s\n", err.Error())

		response := structs.Response{
			Message: fmt.Sprintf("Error en el comando: %s", err.Error()),
			Data:    nil,
			Type:    "system-error",
		}

		utils.SendJSONResponse(w, response, http.StatusBadRequest)
		return
	}

	// Obtener el manager de montajes
	mountManager := mounted.GetMountManager()

	// Verificar si la partición existe antes de desmontar
	partition, exists := mountManager.GetMountedPartition(params.ID)
	if !exists {
		response := structs.Response{
			Message: fmt.Sprintf("Error: No existe ninguna partición montada con el ID '%s'", params.ID),
			Data:    nil,
			Type:    "system-error",
		}

		utils.SendJSONResponse(w, response, http.StatusNotFound)
		return
	}

	// Desmontar la partición
	success := mountManager.UnmountPartition(params.ID)

	if !success {
		response := structs.Response{
			Message: fmt.Sprintf("Error al desmontar la partición con ID '%s'", params.ID),
			Data:    nil,
			Type:    "system-error",
		}

		utils.SendJSONResponse(w, response, http.StatusInternalServerError)
		return
	}

	confirmationMessage := fmt.Sprintf("Partición desmontada exitosamente - ID: %s, Path: %s, Name: %s",
		params.ID, partition.Path, partition.Name)

	response := structs.Response{
		Message: confirmationMessage,
		Data:    params,
		Type:    "system",
	}

	utils.SendJSONResponse(w, response, http.StatusOK)
}
