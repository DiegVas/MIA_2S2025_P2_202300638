package routes

import (
	"fmt"
	"main/Analyzer"
	"main/structs"
	"main/utils"
	"net/http"
)

func handleMkdisk(w http.ResponseWriter, command string) {
	params, err := Analyzer.ParseMkdisk(command)

	if err != nil {
		fmt.Printf("handleMkdisk - error: %s\n", err.Error())

		response := structs.Response{
			Message: fmt.Sprintf("Error en el comando: %s", err.Error()),
			Data:    nil,
			Type:    "system-error",
		}

		utils.SendJSONResponse(w, response, http.StatusBadRequest)
		return
	}

	// ? Crear disco
	err = utils.CreateDisk(params)

	if err != nil {
		response := structs.Response{
			Message: fmt.Sprintf("Error al crear disco: %s", err.Error()),
			Data:    nil,
			Type:    "system-error",
		}

		utils.SendJSONResponse(w, response, http.StatusInternalServerError)
		return
	}

	// ? Registrar disco en archivo ~/disks
	err = utils.RegisterDisk(params)
	if err != nil {
		// Log el error pero no fallar la operación completa
		fmt.Printf("Advertencia: No se pudo registrar el disco en ~/disks: %s\n", err.Error())
	}

	confirmationMessage := fmt.Sprintf("Disco creado exitosamente - Size: %d, Unit: %c, Fit: %s, Path: %s",
		params.Size, params.Unit, params.Fit, params.Path)

	response := structs.Response{
		Message: confirmationMessage,
		Data:    params,
		Type:    "system",
	}

	utils.SendJSONResponse(w, response, http.StatusCreated)
}

func handleDisks(w http.ResponseWriter) {
	disks, err := utils.GetRegisteredDisks()
	if err != nil {
		response := structs.Response{
			Message: fmt.Sprintf("Error al leer discos registrados: %s", err.Error()),
			Data:    nil,
			Type:    "system-error",
		}
		utils.SendJSONResponse(w, response, http.StatusInternalServerError)
		return
	}

	response := structs.Response{
		Message: "Discos registrados:",
		Data:    disks,
		Type:    "system",
	}

	utils.SendJSONResponse(w, response, http.StatusOK)
}
