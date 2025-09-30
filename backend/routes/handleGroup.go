package routes

import (
	"fmt"
	"main/Analyzer"
	"main/provider/users"

	"main/structs"
	"main/utils"
	"net/http"
)

func handleMkgrp(w http.ResponseWriter, command string) {
	params, err := Analyzer.ParseMkgrp(command)
	if err != nil {
		fmt.Printf("handleMkgrp - error: %s\n", err.Error())

		response := structs.Response{
			Message: fmt.Sprintf("Error en el comando: %s", err.Error()),
			Data:    nil,
			Type:    "system-error",
		}

		utils.SendJSONResponse(w, response, http.StatusBadRequest)
		return
	}

	// ! Crear el grupo
	err = users.CreateGroup(params)
	if err != nil {
		fmt.Printf("handleMkgrp - error: %s\n", err.Error())

		response := structs.Response{
			Message: fmt.Sprintf("Error al crear grupo: %s", err.Error()),
			Data:    nil,
			Type:    "system-error",
		}

		utils.SendJSONResponse(w, response, http.StatusInternalServerError)
		return
	}

	confirmationMessage := fmt.Sprintf("Grupo '%s' creado exitosamente", params.Name)

	response := utils.CreateResponse(confirmationMessage, params, "system")

	utils.SendJSONResponse(w, response, http.StatusOK)
}

func handleRmgrp(w http.ResponseWriter, command string) {

	params, err := Analyzer.ParseRmgrp(command)
	if err != nil {
		fmt.Printf("handleRmgrp - error: %s\n", err.Error())

		response := structs.Response{
			Message: fmt.Sprintf("Error en el comando: %s", err.Error()),
			Data:    nil,
			Type:    "system-error",
		}

		utils.SendJSONResponse(w, response, http.StatusBadRequest)
		return
	}

	err = users.RemoveGroup(params)
	if err != nil {
		fmt.Printf("handleRmgrp - error: %s\n", err.Error())

		response := structs.Response{
			Message: fmt.Sprintf("Error al eliminar grupo: %s", err.Error()),
			Data:    nil,
			Type:    "system-error",
		}

		utils.SendJSONResponse(w, response, http.StatusInternalServerError)
		return
	}

	confirmationMessage := fmt.Sprintf("Grupo '%s' eliminado exitosamente", params.Name)
	response := utils.CreateResponse(confirmationMessage, params, "system")

	utils.SendJSONResponse(w, response, http.StatusOK)
}
