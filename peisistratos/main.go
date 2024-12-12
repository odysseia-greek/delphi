package main

import (
	"github.com/odysseia-greek/agora/plato/logging"
	"github.com/odysseia-greek/delphi/peisistratos/architect"
	"log"
	"os"
	"strings"
)

func main() {
	//https://patorjk.com/software/taag/#p=display&f=Crawford2&t=Peisistratos
	logging.System(`
 ____   ___  ____ _____ ____ _____ ______  ____    ____  ______   ___   _____
|    \ /  _]|    / ___/|    / ___/|      ||    \  /    ||      | /   \ / ___/
|  o  )  [_  |  (   \_  |  (   \_ |      ||  D  )|  o  ||      ||     (   \_ 
|   _/    _] |  |\__  | |  |\__  ||_|  |_||    / |     ||_|  |_||  O  |\__  |
|  | |   [_  |  |/  \ | |  |/  \ |  |  |  |    \ |  _  |  |  |  |     |/  \ |
|  | |     | |  |\    | |  |\    |  |  |  |  .  \|  |  |  |  |  |     |\    |
|__| |_____||____|\___||____|\___|  |__|  |__|\_||__|__|  |__|   \___/  \___|
                                                                             
`)
	logging.System(strings.Repeat("~", 37))
	logging.System("\"καὶ Πεισίστρατος μὲν ἐτυράννευε Ἀθηνέων\"")
	logging.System("\"So Pisistratus was sovereign of Athens\"")
	logging.System(strings.Repeat("~", 37))

	logging.System("creating config")

	handler, err := architect.CreateNewConfig()
	if err != nil {
		log.Fatal("death has found me")
	}

	err = handler.InitVault()
	if err != nil {
		log.Print(err.Error())
		os.Exit(1)
	} else {
		os.Exit(0)
	}

}
