package main

import (
	"github.com/odysseia-greek/delphi/peisistratos/app"
	"log"
	"os"
	"strings"
)

func main() {
	//https://patorjk.com/software/taag/#p=display&f=Crawford2&t=Peisistratos
	log.Print(`
 ____   ___  ____ _____ ____ _____ ______  ____    ____  ______   ___   _____
|    \ /  _]|    / ___/|    / ___/|      ||    \  /    ||      | /   \ / ___/
|  o  )  [_  |  (   \_  |  (   \_ |      ||  D  )|  o  ||      ||     (   \_ 
|   _/    _] |  |\__  | |  |\__  ||_|  |_||    / |     ||_|  |_||  O  |\__  |
|  | |   [_  |  |/  \ | |  |/  \ |  |  |  |    \ |  _  |  |  |  |     |/  \ |
|  | |     | |  |\    | |  |\    |  |  |  |  .  \|  |  |  |  |  |     |\    |
|__| |_____||____|\___||____|\___|  |__|  |__|\_||__|__|  |__|   \___/  \___|
                                                                             
`)
	log.Print(strings.Repeat("~", 37))
	log.Print("\"καὶ Πεισίστρατος μὲν ἐτυράννευε Ἀθηνέων\"")
	log.Print("\"So Pisistratus was sovereign of Athens\"")
	log.Print(strings.Repeat("~", 37))

	log.Print("creating config")

	env := os.Getenv("ENV")

	handler, err := app.CreateNewConfig(env)
	if err != nil {
		log.Fatal("death has found me")
	}

	err = handler.InitVault()

	log.Print(err.Error())

	if err != nil {
		os.Exit(1)
	} else {
		os.Exit(0)
	}

}
