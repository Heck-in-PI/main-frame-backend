package v1_common

import (
	"encoding/json"
	"errors"
	"net/http"
	"os/exec"
	"strings"
)

type ErrorMessage struct {
	Error string
}

func JsonResponceHandler(resp http.ResponseWriter, statusCode int, data interface{}) {

	resp.Header().Set("Content-Type", "application/json")
	resp.WriteHeader(statusCode)
	json.NewEncoder(resp).Encode(data)
}

func CommandExec(command string) (string, error) {

	commandParsed := strings.Split(command, " ")
	if len(commandParsed) == 0 {
		return "", errors.New("parsing command error")
	}

	output, err := exec.Command(commandParsed[0], commandParsed[1:]...).Output()
	if err != nil {
		return "", err
	}

	return string(output), nil
}
