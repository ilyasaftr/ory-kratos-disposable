package httpx

import (
	"encoding/json"
	"net/http"

	"github.com/ilyasaftr/ory-kratos-disposable/internal/domain"
)

const contentTypeJSON = "application/json"

func WriteJSON(w http.ResponseWriter, statusCode int, data interface{}) error {
	w.Header().Set("Content-Type", contentTypeJSON)
	w.WriteHeader(statusCode)
	return json.NewEncoder(w).Encode(data)
}

func WriteOryError(w http.ResponseWriter, statusCode int, message string) error {
	resp := domain.OryWebhookResponse{
		Messages: []domain.MessageGroup{
			{
				InstancePtr: "#/",
				Messages: []domain.Message{
					{
						ID:   statusCode,
						Text: message,
						Type: "error",
					},
				},
			},
		},
	}

	return WriteJSON(w, statusCode, resp)
}
