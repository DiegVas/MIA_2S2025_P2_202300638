package routes

import (
	"fmt"
	"main/Analyzer"
	diskmanager "main/diskManager"
	"main/structs"
	"main/utils"
	"net/http"
)

func handleFdisk(w http.ResponseWriter, command string) {
	params, err := Analyzer.ParseFdisk(command)
	if err != nil {
		fmt.Printf("handleFdisk - error: %s\n", err.Error())

		response := structs.Response{
			Message: fmt.Sprintf("Error en el comando: %s", err.Error()),
			Data:    nil,
			Type:    "system-error",
		}

		utils.SendJSONResponse(w, response, http.StatusBadRequest)
		return
	}

	// Determinar el tipo de operación
	if params.Delete != "" {
		// Operación de eliminación
		handleDeletePartition(w, params)
		return
	}

	if params.Add != 0 {
		// Operación de agregar/quitar espacio
		handleAddPartitionSpace(w, params)
		return
	}

	// Operación de creación de partición (comportamiento original)
	handleCreatePartition(w, params)
}

func handleCreatePartition(w http.ResponseWriter, params *structs.FdiskParams) {
	err := diskmanager.CreatePartition(params)
	if err != nil {
		response := structs.Response{
			Message: fmt.Sprintf("Error al crear partición: %s", err.Error()),
			Data:    nil,
			Type:    "system-error",
		}

		utils.SendJSONResponse(w, response, http.StatusInternalServerError)
		return
	}

	// Registrar partición en el disco correspondiente
	err = diskmanager.RegisterPartition(params.Path, params.Name)
	if err != nil {
		// Log el error pero no fallar la operación completa
		fmt.Printf("Advertencia: No se pudo registrar la partición en ~/disks: %s\n", err.Error())
	}

	confirmationMessage := fmt.Sprintf("Partición creada exitosamente - Name: %s, Size: %d, Type: %c, Path: %s",
		params.Name, params.Size, params.Type, params.Path)

	response := structs.Response{
		Message: confirmationMessage,
		Data:    params,
		Type:    "system",
	}

	utils.SendJSONResponse(w, response, http.StatusCreated)
}

func handleDeletePartition(w http.ResponseWriter, params *structs.FdiskParams) {
	// Implementar la lógica de eliminación
	err := diskmanager.DeletePartition(params)

	if err != nil {
		response := structs.Response{
			Message: fmt.Sprintf("Error al eliminar partición: %s", err.Error()),
			Data:    nil,
			Type:    "system-error",
		}

		utils.SendJSONResponse(w, response, http.StatusInternalServerError)
		return
	}

	confirmationMessage := fmt.Sprintf("Partición '%s' eliminada exitosamente (%s)",
		params.Name, params.Delete)

	response := structs.Response{
		Message: confirmationMessage,
		Data:    params,
		Type:    "system",
	}

	utils.SendJSONResponse(w, response, http.StatusOK)
}

func handleAddPartitionSpace(w http.ResponseWriter, params *structs.FdiskParams) {
	// Implementar la lógica de agregar/quitar espacio
	err := diskmanager.ModifyPartitionSize(params)

	if err != nil {
		response := structs.Response{
			Message: fmt.Sprintf("Error al modificar el tamaño de la partición: %s", err.Error()),
			Data:    nil,
			Type:    "system-error",
		}

		utils.SendJSONResponse(w, response, http.StatusInternalServerError)
		return
	}

	operation := "agregado"
	if params.Add < 0 {
		operation = "quitado"
	}

	confirmationMessage := fmt.Sprintf("Espacio %s exitosamente a la partición '%s' - Cantidad: %d %c",
		operation, params.Name, params.Add, params.Unit)

	response := structs.Response{
		Message: confirmationMessage,
		Data:    params,
		Type:    "system",
	}

	utils.SendJSONResponse(w, response, http.StatusOK)
}
