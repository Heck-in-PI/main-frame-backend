package v1_common

import (
	"encoding/json"
	"net/http"
)

type ErrorMessage struct {
	Error string
}

func JsonResponceHandler(resp http.ResponseWriter, statusCode int, data interface{}) {

	resp.Header().Set("Content-Type", "application/json")
	resp.WriteHeader(statusCode)
	json.NewEncoder(resp).Encode(data)
}
