package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/odysseia-greek/delphi/solon/app"
	"github.com/odysseia-greek/delphi/solon/config"
	plato "github.com/odysseia-greek/plato/config"
	"github.com/odysseia-greek/plato/logging"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

const standardPort = ":5443"

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = standardPort
	}
	//https://patorjk.com/software/taag/#p=display&f=Crawford2&t=SOLON
	logging.System("\n  _____  ___   _       ___   ____  \n / ___/ /   \\ | |     /   \\ |    \\ \n(   \\_ |     || |    |     ||  _  |\n \\__  ||  O  || |___ |  O  ||  |  |\n /  \\ ||     ||     ||     ||  |  |\n \\    ||     ||     ||     ||  |  |\n  \\___| \\___/ |_____| \\___/ |__|__|\n                                   \n")
	logging.System("\"αὐτοὶ γὰρ οὐκ οἷοί τε ἦσαν αὐτὸ ποιῆσαι Ἀθηναῖοι: ὁρκίοισι γὰρ μεγάλοισι κατείχοντο δέκα ἔτεα χρήσεσθαι νόμοισι τοὺς ἄν σφι Σόλων θῆται.\"")
	logging.System("\"since the Athenians themselves could not do that, for they were bound by solemn oaths to abide for ten years by whatever laws Solon should make.\"")
	logging.System("starting up.....")
	logging.Debug("starting up and getting env variables")

	env := os.Getenv("ENV")

	solonConfig, err := config.CreateNewConfig(env)
	if err != nil {
		logging.Error(err.Error())
		log.Fatal("death has found me")
	}

	logging.System("creating tracing user at startup")
	err = solonConfig.CreateTracingUser(false)

	srv := app.InitRoutes(*solonConfig)
	logging.System(fmt.Sprintf("%s : %v", "TLS enabled", solonConfig.TLSEnabled))
	logging.System(fmt.Sprintf("%s : %s", "running on port", port))

	if solonConfig.TLSEnabled {
		rootPath := os.Getenv("CERT_ROOT")
		if rootPath == "" {
			logging.Error("rootpath is empty no certs can be loaded")
		}
		fp := filepath.Join(rootPath, "solon", "tls.pem")
		caFromFile, _ := ioutil.ReadFile(fp)
		ca := x509.NewCertPool()
		ca.AppendCertsFromPEM(caFromFile)
		httpsServer := createTlSConfig(port, ca, srv)
		overwrite := os.Getenv("TESTOVERWRITE")
		var testOverwrite bool
		if overwrite != "" {
			testOverwrite = true
		}

		logging.Debug("loading cert files from mount")
		certPath, keyPath := plato.RetrieveCertPathLocally(testOverwrite, "solon")
		err = httpsServer.ListenAndServeTLS(certPath, keyPath)
		if err != nil {
			log.Fatal("death has found me")
		}
	} else {
		err = http.ListenAndServe(port, srv)
		if err != nil {
			panic(err)
		}
	}
}

func createTlSConfig(port string, ca *x509.CertPool, server *mux.Router) *http.Server {
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
		//ClientAuth: tls.RequireAndVerifyClientCert,
		ClientCAs: ca,
	}

	return &http.Server{
		Addr:         port,
		Handler:      server,
		TLSConfig:    cfg,
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler), 0),
	}
}
