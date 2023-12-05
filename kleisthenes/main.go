package main

import (
	"github.com/odysseia-greek/agora/plato/logging"
	"github.com/odysseia-greek/delphi/kleisthenes/app"
	"log"
	"os"
	"strings"
)

func main() {
	//https://patorjk.com/software/taag/#p=display&f=Crawford2&t=Kleisthenes
	logging.System(`
 __  _  _        ___  ____ _____ ______  __ __    ___  ____     ___  _____
|  |/ ]| |      /  _]|    / ___/|      ||  |  |  /  _]|    \   /  _]/ ___/
|  ' / | |     /  [_  |  (   \_ |      ||  |  | /  [_ |  _  | /  [_(   \_ 
|    \ | |___ |    _] |  |\__  ||_|  |_||  _  ||    _]|  |  ||    _]\__  |
|     ||     ||   [_  |  |/  \ |  |  |  |  |  ||   [_ |  |  ||   [_ /  \ |
|  .  ||     ||     | |  |\    |  |  |  |  |  ||     ||  |  ||     |\    |
|__|\_||_____||_____||____|\___|  |__|  |__|__||_____||__|__||_____| \___|
                                                                          
`)
	logging.System(strings.Repeat("~", 37))
	logging.System("\"ὀστρακισμός\"")
	logging.System("\"ostracism, introduced by Kleisthenes\"")
	logging.System(strings.Repeat("~", 37))

	logging.System("creating config")

	env := os.Getenv("ENV")

	handler, err := app.CreateNewConfig(env)
	if err != nil {
		log.Fatal("death has found me")
	}

	if err := handler.Create(); err != nil {
		logging.Error(err.Error())
		os.Exit(1)
	}

	logging.Info("setup complete")
	os.Exit(0)
}
