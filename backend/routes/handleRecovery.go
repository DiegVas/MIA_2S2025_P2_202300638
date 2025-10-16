package routes

import (
	"fmt"
	"main/Analyzer"
	diskmanager "main/diskManager"
	"main/provider/mounted"
	"main/structs"
	"main/utils"
	"main/utils/journaling"
	"net/http"
)

func handleRecovery(w http.ResponseWriter, command string) {
	fmt.Println("handleRecovery - command:", command)
	params, err := Analyzer.ParseRecovery(command)
	if err != nil {
		fmt.Printf("handleRecovery - error: %s\n", err.Error())

		response := structs.Response{
			Message: fmt.Sprintf("Error en el comando: %s", err.Error()),
			Data:    nil,
			Type:    "system-error",
		}

		utils.SendJSONResponse(w, response, http.StatusBadRequest)
		return
	}

	// Buscar la partición montada por ID
	mountManager := mounted.GetMountManager()
	partition, found := mountManager.GetMountedPartition(params.ID)
	if !found {
		response := structs.Response{
			Message: fmt.Sprintf("No se encontró una partición montada con ID: %s", params.ID),
			Data:    nil,
			Type:    "system-error",
		}

		utils.SendJSONResponse(w, response, http.StatusBadRequest)
		return
	}

	// Ejecutar la recuperación del sistema de archivos EXT3
	err = diskmanager.RecoverFileSystem(partition.Path, partition.Name)
	if err != nil {
		response := structs.Response{
			Message: fmt.Sprintf("Error al recuperar el sistema de archivos: %s", err.Error()),
			Data:    nil,
			Type:    "system-error",
		}

		utils.SendJSONResponse(w, response, http.StatusInternalServerError)
		return
	}

	confirmationMessage := fmt.Sprintf("Sistema de archivos EXT3 recuperado exitosamente para la partición ID: %s", params.ID)

	response := structs.Response{
		Message: confirmationMessage,
		Data:    params,
		Type:    "system",
	}

	utils.SendJSONResponse(w, response, http.StatusOK)
}

func handleLoss(w http.ResponseWriter, command string) {
	fmt.Println("handleLoss - command:", command)
	params, err := Analyzer.ParseLoss(command)
	if err != nil {
		fmt.Printf("handleLoss - error: %s\n", err.Error())

		response := structs.Response{
			Message: fmt.Sprintf("Error en el comando: %s", err.Error()),
			Data:    nil,
			Type:    "system-error",
		}

		utils.SendJSONResponse(w, response, http.StatusBadRequest)
		return
	}

	// Buscar la partición montada por ID
	mountManager := mounted.GetMountManager()
	partition, found := mountManager.GetMountedPartition(params.ID)
	if !found {
		response := structs.Response{
			Message: fmt.Sprintf("No se encontró una partición montada con ID: %s", params.ID),
			Data:    nil,
			Type:    "system-error",
		}

		utils.SendJSONResponse(w, response, http.StatusBadRequest)
		return
	}

	// Simular la pérdida del sistema de archivos EXT3
	err = diskmanager.SimulateFileSystemLoss(partition.Path, partition.Name)
	if err != nil {
		response := structs.Response{
			Message: fmt.Sprintf("Error al simular la pérdida del sistema de archivos: %s", err.Error()),
			Data:    nil,
			Type:    "system-error",
		}

		utils.SendJSONResponse(w, response, http.StatusInternalServerError)
		return
	}

	confirmationMessage := fmt.Sprintf("Pérdida del sistema de archivos EXT3 simulada exitosamente para la partición ID: %s", params.ID)

	response := structs.Response{
		Message: confirmationMessage,
		Data:    params,
		Type:    "system",
	}

	utils.SendJSONResponse(w, response, http.StatusOK)
}

func handleJournaling(w http.ResponseWriter, command string) {
	fmt.Println("handleJournaling - command:", command)
	params, err := Analyzer.ParseJournaling(command)
	if err != nil {
		fmt.Printf("handleJournaling - error: %s\n", err.Error())

		response := structs.Response{
			Message: fmt.Sprintf("Error en el comando: %s", err.Error()),
			Data:    nil,
			Type:    "system-error",
		}

		utils.SendJSONResponse(w, response, http.StatusBadRequest)
		return
	}

	// Buscar la partición montada por ID
	mountManager := mounted.GetMountManager()
	_, found := mountManager.GetMountedPartition(params.ID)
	if !found {
		response := structs.Response{
			Message: fmt.Sprintf("No se encontró una partición montada con ID: %s", params.ID),
			Data:    nil,
			Type:    "system-error",
		}

		utils.SendJSONResponse(w, response, http.StatusBadRequest)
		return
	}

	// Obtener información del journaling
	journalData := journaling.GetJournalingInfo(params.ID)

	response := structs.Response{
		Message: fmt.Sprintf("Información de journaling para la partición ID: %s", params.ID),
		Data:    journalData,
		Type:    "journaling",
	}

	utils.SendJSONResponse(w, response, http.StatusOK)
}
