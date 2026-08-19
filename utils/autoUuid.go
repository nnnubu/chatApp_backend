package utils

import "github.com/google/uuid"

func GenAutoUuid() string {
	return uuid.Must(uuid.NewV7()).String()
}
