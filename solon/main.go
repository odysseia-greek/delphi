package main

import (
	"crypto/tls"
	"crypto/x509"
	"github.com/gorilla/mux"
	"github.com/kpango/glg"
	"github.com/odysseia-greek/delphi/solon/app"
	"github.com/odysseia-greek/delphi/solon/config"
	plato "github.com/odysseia-greek/plato/config"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
)

const standardPort = ":5443"

func init() {
	errlog := glg.FileWriter("/tmp/error.log", 0666)
	defer errlog.Close()

	glg.Get().
		SetMode(glg.BOTH).
		AddLevelWriter(glg.ERR, errlog)
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = standardPort
	}
	//https://patorjk.com/software/taag/#p=display&f=Crawford2&t=SOLON
	glg.Info("\n  _____  ___   _       ___   ____  \n / ___/ /   \\ | |     /   \\ |    \\ \n(   \\_ |     || |    |     ||  _  |\n \\__  ||  O  || |___ |  O  ||  |  |\n /  \\ ||     ||     ||     ||  |  |\n \\    ||     ||     ||     ||  |  |\n  \\___| \\___/ |_____| \\___/ |__|__|\n                                   \n")
	glg.Info("\"αὐτοὶ γὰρ οὐκ οἷοί τε ἦσαν αὐτὸ ποιῆσαι Ἀθηναῖοι: ὁρκίοισι γὰρ μεγάλοισι κατείχοντο δέκα ἔτεα χρήσεσθαι νόμοισι τοὺς ἄν σφι Σόλων θῆται.\"")
	glg.Info("\"since the Athenians themselves could not do that, for they were bound by solemn oaths to abide for ten years by whatever laws Solon should make.\"")
	glg.Info("starting up.....")
	glg.Debug("starting up and getting env variables")

	env := os.Getenv("ENV")

	solonConfig, err := config.CreateNewConfig(env)
	if err != nil {
		glg.Error(err)
		glg.Fatal("death has found me")
	}

	glg.Info("creating tracing user at startup")
	err = solonConfig.CreateTracingUser()

	srv := app.InitRoutes(*solonConfig)
	glg.Infof("%s : %v", "TLS enabled", solonConfig.TLSEnabled)
	glg.Infof("%s : %s", "running on port", port)

	if solonConfig.TLSEnabled {
		rootPath := os.Getenv("CERT_ROOT")
		if rootPath == "" {
			glg.Error("rootpath is empty no certs can be loaded")
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

		glg.Debug("loading cert files from mount")
		certPath, keyPath := plato.RetrieveCertPathLocally(testOverwrite, "solon")
		err = httpsServer.ListenAndServeTLS(certPath, keyPath)
		if err != nil {
			glg.Fatal(err)
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
