package proxy

import "github.com/google/uuid"

func generateRequestID() string {
	return uuid.New().String()
}
