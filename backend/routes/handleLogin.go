package routes

import (
	"fmt"
	"main/Analyzer"
	"main/provider/users"
	"main/structs"
	"main/utils"
	"net/http"
)

func handleLogin(w http.ResponseWriter, command string) {
	params, err := Analyzer.ParseLogin(command)
	if err != nil {
		response := structs.Response{
			Message: fmt.Sprintf("Error en el comando: %s", err.Error()),
			Data:    nil,
			Type:    "system-error",
		}

		utils.SendJSONResponse(w, response, http.StatusBadRequest)
		return
	}

	err = users.LoginUser(params)
	if err != nil {
		response := structs.Response{
			Message: err.Error(),
			Data:    nil,
			Type:    "auth-error",
		}

		utils.SendJSONResponse(w, response, http.StatusUnauthorized)
		return
	}

	response := utils.CreateResponse(
		fmt.Sprintf("Sesión iniciada correctamente como '%s' en la partición '%s'", params.User, params.ID),
		params,
		"system",
	)

	utils.SendJSONResponse(w, response, http.StatusOK)
}

func handleLogout(w http.ResponseWriter, command string) {
	err := Analyzer.ParseLogout(command)
	if err != nil {
		response := structs.Response{
			Message: err.Error(),
			Data:    nil,
			Type:    "system-error",
		}

		utils.SendJSONResponse(w, response, http.StatusBadRequest)
		return
	}
	err = users.LogoutUser()
	if err != nil {
		response := structs.Response{
			Message: err.Error(),
			Data:    nil,
			Type:    "auth-error",
		}

		utils.SendJSONResponse(w, response, http.StatusUnauthorized)
		return
	}
	response := structs.Response{
		Message: "Sesión cerrada correctamente",
		Data:    nil,
		Type:    "auth-success",
	}

	utils.SendJSONResponse(w, response, http.StatusOK)
}

func handleSesion(w http.ResponseWriter, command string) {
	if command != "sesion" {
		response := structs.Response{
			Message: "Comando no reconocido. El comando 'sesion' es el único permitido.",
			Data:    nil,
			Type:    "system-error",
		}

		utils.SendJSONResponse(w, response, http.StatusBadRequest)
		return
	}

	user := structs.GetSession()
	if !user.Active {
		response := structs.Response{
			Message: "No hay sesión activa",
			Data:    nil,
			Type:    "auth-error",
		}
		utils.SendJSONResponse(w, response, http.StatusUnauthorized)
		return
	}

	response := structs.Response{
		Message: "usuario= " + user.User,
		Data:    user,
		Type:    "auth-success",
	}
	utils.SendJSONResponse(w, response, http.StatusOK)
}
