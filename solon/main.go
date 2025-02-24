package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/odysseia-greek/agora/plato/logging"
	"github.com/odysseia-greek/agora/plato/tlsmanager"
	"github.com/odysseia-greek/delphi/solon/lawgiver"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const defaultPort = ":5443"
const SolonService string = "solon"

var currentTLSConfig *tls.Config

func main() {
	port := getEnv("PORT", defaultPort)

	logBanner()

	ctx := context.Background()

	// Initialize Solon handler
	solonHandler, err := lawgiver.CreateNewConfig(ctx)
	if err != nil {
		logging.Error(fmt.Sprintf("Failed to initialize Solon handler: %v", err))
		log.Fatal("Startup failure")
	}

	// Setup server
	srv := lawgiver.InitRoutes(solonHandler)
	logging.System(fmt.Sprintf("TLS enabled: %v", solonHandler.TLSEnabled))
	logging.System(fmt.Sprintf("Running on port: %s", port))

	go func() {
		err := solonHandler.StartWatching()
		if err != nil {
			logging.Error(fmt.Sprintf("Failed to start watching deployments and pods: %v", err))
		}
	}()

	if solonHandler.TLSEnabled {
		startTLSServer(port, srv)
	} else {
		startHTTPServer(port, srv)
	}
}

func logBanner() {
	logging.System(`
  _____  ___   _       ___   ____  
 / ___/ /   \ | |     /   \ |    \ 
(   \_ |     || |    |     ||  _  |
 \__  ||  O  || |___ |  O  ||  |  |
 /  \ ||     ||     ||     ||  |  |
 \    ||     ||     ||     ||  |  |
  \___| \___/ |_____| \___/ |__|__|
`)
	logging.System("\"αὐτοὶ γὰρ οὐκ οἷοί τε ἦσαν αὐτὸ ποιῆσαι Ἀθηναῖοι: ὁρκίοισι γὰρ μεγάλοισι κατείχοντο δέκα ἔτεα χρήσεσθαι νόμοισι τοὺς ἄν σφι Σόλων θῆται.\"")
	logging.System("\"Since the Athenians themselves could not do that, for they were bound by solemn oaths to abide for ten years by whatever laws Solon should make.\"")
	logging.System("Starting up...")
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func startHTTPServer(port string, srv *mux.Router) {
	logging.System("Starting HTTP server...")
	if err := http.ListenAndServe(port, srv); err != nil {
		logging.Error(fmt.Sprintf("HTTP server error: %v", err))
		log.Fatal("Server shutdown")
	}
}

func startTLSServer(port string, srv *mux.Router) {
	gracePeriod := 1 * time.Hour
	pollInterval := 5 * time.Minute

	rootPath := os.Getenv("CERT_ROOT")
	if rootPath == "" {
		logging.Error(fmt.Sprintf("CERT_ROOT environment variable is empty. Defaulting to: %s", tlsmanager.DefaultCertRoot))
		rootPath = tlsmanager.DefaultCertRoot
	}

	// Load CA certificate for self-signed certificate validation
	caPath := filepath.Join(rootPath, SolonService, "tls.pem")
	caFromFile, err := os.ReadFile(caPath)
	if err != nil {
		log.Fatalf("Failed to read CA certificate: %v", err)
	}

	ca := x509.NewCertPool()
	if !ca.AppendCertsFromPEM(caFromFile) {
		log.Fatalf("Failed to append CA certificate from: %s", caPath)
	}

	// Initialize the TLSManager
	tlsManager := tlsmanager.NewTLSManager(SolonService, rootPath, gracePeriod)

	// Load the initial certificates
	if err := tlsManager.LoadCertificates(); err != nil {
		log.Fatalf("Failed to load initial certificates: %v", err)
	}

	// Start watching for certificate changes
	tlsManager.WatchCertificates(pollInterval)

	// Create and configure the HTTPS server
	server := &http.Server{
		Addr:    port,
		Handler: srv,
		TLSConfig: &tls.Config{
			GetConfigForClient: func(clientHello *tls.ClientHelloInfo) (*tls.Config, error) {
				return tlsManager.GetTLSConfig(), nil
			},
		},
	}

	logging.System("Starting TLS server...")
	if err := server.ListenAndServeTLS("", ""); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
