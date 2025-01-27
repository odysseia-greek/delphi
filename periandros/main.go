package main

import (
	"fmt"
	"github.com/odysseia-greek/agora/plato/logging"
	"github.com/odysseia-greek/delphi/periandros/initiator"
	"log"
	"os"
	"strings"
	"time"
)

func main() {
	//https://patorjk.com/software/taag/#p=display&f=Crawford2&t=PERIANDROS
	logging.System("\n ____   ___  ____   ____   ____  ____   ___    ____   ___   _____\n|    \\ /  _]|    \\ |    | /    ||    \\ |   \\  |    \\ /   \\ / ___/\n|  o  )  [_ |  D  ) |  | |  o  ||  _  ||    \\ |  D  )     (   \\_ \n|   _/    _]|    /  |  | |     ||  |  ||  D  ||    /|  O  |\\__  |\n|  | |   [_ |    \\  |  | |  _  ||  |  ||     ||    \\|     |/  \\ |\n|  | |     ||  .  \\ |  | |  |  ||  |  ||     ||  .  \\     |\\    |\n|__| |_____||__|\\_||____||__|__||__|__||_____||__|\\_|\\___/  \\___|\n                                                                 \n")
	logging.System(strings.Repeat("~", 37))
	logging.System("\"Περίανδρος δὲ ἦν Κυψέλου παῖς οὗτος ὁ τῷ Θρασυβούλῳ τὸ χρηστήριον μηνύσας· ἐτυράννευε δὲ ὁ Περίανδρος Κορίνθου\"")
	logging.System("\"Periander, who disclosed the oracle's answer to Thrasybulus, was the son of Cypselus, and sovereign of Corinth\"")
	logging.System(strings.Repeat("~", 37))

	logging.Debug("creating config")

	duration := 1 * time.Second
	timeOut := 5 * time.Minute

	handler, err := initiator.CreateNewConfig(duration, timeOut)
	if err != nil {
		log.Fatal("death has found me")
	}

	created, err := handler.CreateUser()
	if err != nil {
		logging.Error("an error occurred during creation of user")
		os.Exit(1)
	}

	logging.Info(fmt.Sprintf("created user: %s %v", handler.SolonCreationRequest.Username, created))
}
