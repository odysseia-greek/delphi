package main

import (
	"fmt"
	"github.com/odysseia-greek/agora/plato/logging"
	"github.com/odysseia-greek/delphi/drakon/legislator"
	"log"
	"os"
	"strings"
)

func main() {
	//https://patorjk.com/software/taag/#p=display&f=Crawford2&t=PERIANDROS
	logging.System(`
 ___    ____    ____  __  _   ___   ____  
|   \  |    \  /    ||  |/ ] /   \ |    \ 
|    \ |  D  )|  o  ||  ' / |     ||  _  |
|  D  ||    / |     ||    \ |  O  ||  |  |
|     ||    \ |  _  ||     ||     ||  |  |
|     ||  .  \|  |  ||  .  ||     ||  |  |
|_____||__|\_||__|__||__|\_| \___/ |__|__|
                                          
`)
	logging.System(strings.Repeat("~", 37))
	logging.System("\"ἐν τοίνυν τοῖς περὶ τούτων νόμοις ὁ Δράκων φοβερὸν κατασκευάζων καὶ δεινὸν τό τινʼ αὐτόχειρʼ ἄλλον ἄλλου γίγνεσθαι\"")
	logging.System("\"Now Draco, in this group of laws, marked the terrible wickedness of homicide by banning the offender from the lustral water\"")
	logging.System(strings.Repeat("~", 37))

	logging.Debug("creating config")

	env := os.Getenv("ENV")

	handler, err := legislator.CreateNewConfig(env)
	if err != nil {
		log.Fatal("death has found me")
	}

	created, err := handler.CreateRoles()
	if err != nil {
		logging.Error("an error occurred during creation of roles")
		logging.Error(err.Error())
		os.Exit(1)
	}

	logging.Info(fmt.Sprintf("all roles created: %v", created))
}
