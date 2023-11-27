package main

import (
	"crypto/tls"
	"fmt"
	"github.com/odysseia-greek/delphi/perikles/app"
	"github.com/odysseia-greek/delphi/perikles/config"
	"github.com/odysseia-greek/plato/logging"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const (
	standardPort = "4443"
	crtFileName  = "tls.crt"
	keyFileName  = "tls.key"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = standardPort
	}
	//https://patorjk.com/software/taag/#p=display&f=Crawford2&t=PERIKLES
	logging.System("\n ____   ___  ____   ____  __  _  _        ___  _____\n|    \\ /  _]|    \\ |    ||  |/ ]| |      /  _]/ ___/\n|  o  )  [_ |  D  ) |  | |  ' / | |     /  [_(   \\_ \n|   _/    _]|    /  |  | |    \\ | |___ |    _]\\__  |\n|  | |   [_ |    \\  |  | |     ||     ||   [_ /  \\ |\n|  | |     ||  .  \\ |  | |  .  ||     ||     |\\    |\n|__| |_____||__|\\_||____||__|\\_||_____||_____| \\___|\n                                                    \n")
	logging.System(strings.Repeat("~", 37))
	logging.System("\"τόν γε σοφώτατον οὐχ ἁμαρτήσεται σύμβουλον ἀναμείνας χρόνον.\"")
	logging.System("\"he would yet do full well to wait for that wisest of all counsellors, Time.\"")
	logging.System(strings.Repeat("~", 37))

	env := os.Getenv("ENV")

	periklesConfig, err := config.CreateNewConfig(env)
	if err != nil {
		log.Fatal("death has found me")
	}

	handler := app.PeriklesHandler{Config: periklesConfig}

	logging.Debug("init for CA started...")
	err = handler.Config.Cert.InitCa()
	if err != nil {
		log.Fatal("death has found me")
	}

	logging.Debug("CA created")

	logging.Debug("creating CRD...")
	created, err := handler.Config.Kube.V1Alpha1().ServiceMapping().CreateInCluster()
	if err != nil {
		logging.Error(err.Error())
	}

	if created {
		logging.Debug("CRD created")
	} else {
		logging.Debug("CRD not created, it might already exist")
	}

	_, err = handler.Config.Kube.V1Alpha1().ServiceMapping().Get(periklesConfig.CrdName)
	if err != nil {
		mapping, err := handler.Config.Kube.V1Alpha1().ServiceMapping().Parse(nil, periklesConfig.CrdName, periklesConfig.Namespace)
		if err != nil {
			logging.Error(err.Error())
		}

		createdCrd, err := handler.Config.Kube.V1Alpha1().ServiceMapping().Create(mapping)
		if err != nil {
			logging.Error(err.Error())
		}

		logging.Debug(fmt.Sprintf("created mapping %s", createdCrd.Name))

	}

	logging.Debug("init routes")
	srv := app.InitRoutes(*periklesConfig)

	cfg := &tls.Config{
		MinVersion:               tls.VersionTLS12,
		CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
		PreferServerCipherSuites: true,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
			tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_RSA_WITH_AES_256_CBC_SHA,
		},
	}

	logging.Debug("setting up server with https")

	httpsServer := &http.Server{
		Addr:         fmt.Sprintf(":%s", port),
		Handler:      srv,
		TLSConfig:    cfg,
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler), 0),
	}

	logging.Debug("loading cert files from mount")
	certFile := filepath.Join(periklesConfig.TLSFiles, crtFileName)
	keyFile := filepath.Join(periklesConfig.TLSFiles, keyFileName)

	err = httpsServer.ListenAndServeTLS(certFile, keyFile)
	if err != nil {
		log.Fatal(err)
	}
}
