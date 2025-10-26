package utils

import (
	"net/http"
	"os"
)

func EnableCORS(w http.ResponseWriter, r *http.Request) {
	// Lista de orígenes permitidos
	allowedOrigins := []string{
		"http://localhost:5173",
		"http://localhost:3000",
		"http://localhost:4173",
		"https://tu-dominio.com", // Agrega tu dominio de producción aquí
	}

	origin := r.Header.Get("Origin")

	// Verificar si el origen está en la lista permitida
	isAllowed := false
	for _, allowedOrigin := range allowedOrigins {
		if origin == allowedOrigin {
			isAllowed = true
			break
		}
	}

	// Si está permitido, usar el origen específico; si no, usar el primer origen como fallback
	if isAllowed {
		w.Header().Set("Access-Control-Allow-Origin", origin)
	} else {
		// Para desarrollo, permitir cualquier origen
		envOrigin := os.Getenv("ALLOWED_ORIGIN")
		if envOrigin == "" {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		} else {
			w.Header().Set("Access-Control-Allow-Origin", envOrigin)
		}
	}
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")

	// Solo permitir credenciales si no estamos usando "*"
	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		w.Header().Set("Access-Control-Allow-Credentials", "true")
	}

	w.Header().Set("Access-Control-Max-Age", "86400")
}

func CORSMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		EnableCORS(w, r)

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next(w, r)
	}
}
