package handler

import (
	"net/http"

	"github.com/divord97/ccc/pkg/response"
)

func Health(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
