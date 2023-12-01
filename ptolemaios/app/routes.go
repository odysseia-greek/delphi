package app

import (
	"github.com/odysseia-greek/delphi/ptolemaios/config"
)

func CreateHandler(config *config.Config) *PtolemaiosHandler {
	handler := PtolemaiosHandler{Config: config}
	return &handler
}
