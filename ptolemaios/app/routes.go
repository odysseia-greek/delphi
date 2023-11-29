package app

import (
	"github.com/odysseia-greek/agora/plato/logging"
	"github.com/odysseia-greek/delphi/ptolemaios/config"
	"os"
	"time"
)

func CreateHandler(config *config.Config) *PtolemaiosHandler {
	handler := PtolemaiosHandler{Config: config, Duration: time.Second * 10}

	if config.RunOnce {
		go func() {
			jobExit := make(chan bool, 1)
			go handler.CheckForJobExit(jobExit)

			select {

			case <-jobExit:
				logging.Debug("exiting because of condition")
				os.Exit(0)
			}
		}()
	}

	return &handler
}
