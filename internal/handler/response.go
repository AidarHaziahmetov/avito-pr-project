package handler

import (
	"net/http"

	"github.com/go-chi/render"
)

// RespondWithJSON отправляет JSON ответ с указанным статус кодом
func RespondWithJSON(w http.ResponseWriter, r *http.Request, statusCode int, data interface{}) {
	render.Status(r, statusCode)
	render.JSON(w, r, data)
}
