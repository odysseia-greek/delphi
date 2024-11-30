package main

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/odysseia-greek/agora/plato/logging"
	"github.com/odysseia-greek/delphi/solon/lawgiver"
	"io"
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
	rootPath := os.Getenv("CERT_ROOT")
	if rootPath == "" {
		logging.Error("CERT_ROOT environment variable is empty. No certificates can be loaded.")
		log.Fatal("Server startup failed")
	}

	// Load CA certificate
	fp := filepath.Join(rootPath, "solon", "tls.pem")
	caFromFile, err := os.ReadFile(fp)
	if err != nil {
		logging.Error(fmt.Sprintf("Failed to read CA certificate: %v", err))
		log.Fatal("Server startup failed")
	}

	ca := x509.NewCertPool()
	ca.AppendCertsFromPEM(caFromFile)

	// Create TLS server configuration
	httpsServer := createTLSConfig(port, ca, srv)

	// Watch for certificate changes
	certPath, keyPath := lawgiver.RetrieveCertPathLocally(SolonService)
	interval := 3 * time.Second
	go watchCertificates(certPath, keyPath, interval)

	// Start HTTPS server
	logging.System("Starting HTTPS server with TLS...")
	if err := httpsServer.ListenAndServeTLS(certPath, keyPath); err != nil {
		logging.Error(fmt.Sprintf("HTTPS server error: %v", err))
		log.Fatal("Server shutdown")
	}
}

func createTLSConfig(port string, ca *x509.CertPool, server *mux.Router) *http.Server {
	// TLS configuration dynamically fetched for every client connection
	cfg := &tls.Config{
		MinVersion:       tls.VersionTLS12,
		CurvePreferences: []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
			tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_RSA_WITH_AES_256_CBC_SHA,
		},
		ClientCAs: ca,
		GetConfigForClient: func(clientHello *tls.ClientHelloInfo) (*tls.Config, error) {
			// Dynamically serve the current TLS configuration
			return currentTLSConfig, nil
		},
	}

	// Set initial TLS configuration
	currentTLSConfig = cfg

	return &http.Server{
		Addr:         port,
		Handler:      server,
		TLSConfig:    cfg, // Attach the dynamic TLS config
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler), 0),
	}
}

func loadCertificates(certPath, keyPath string) {
	// Load the new certificate
	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		logging.Error(fmt.Sprintf("Failed to load certificate: %v", err))
		return
	}

	// Copy the current configuration and add the new certificate
	oldTLSConfig := currentTLSConfig.Clone()
	newCertificates := append(oldTLSConfig.Certificates, cert)

	// Create the new TLS configuration with both old and new certificates
	newTLSConfig := &tls.Config{
		Certificates:     newCertificates,
		MinVersion:       tls.VersionTLS12,
		CurvePreferences: []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
			tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_RSA_WITH_AES_256_CBC_SHA,
		},
		GetCertificate: func(clientHello *tls.ClientHelloInfo) (*tls.Certificate, error) {
			for _, cert := range newCertificates {
				return &cert, nil
			}
			return nil, fmt.Errorf("no certificate available")
		},
	}

	// Update the global TLS configuration to include both sets of certificates
	currentTLSConfig = newTLSConfig

	if len(oldTLSConfig.Certificates) > 0 {
		// Remove the old certificate after the grace period
		logging.System("TLS configuration updated with the new certificate. Grace period started.")
		go func(oldCert tls.Certificate) {
			time.Sleep(2 * time.Minute)

			// Filter out the old certificate
			remainingCertificates := []tls.Certificate{}
			for _, cert := range currentTLSConfig.Certificates {
				if !certEqual(cert, oldCert) {
					remainingCertificates = append(remainingCertificates, cert)
				}
			}

			// Update the TLS configuration to only use the new certificate
			currentTLSConfig = &tls.Config{
				Certificates: remainingCertificates,
				GetCertificate: func(clientHello *tls.ClientHelloInfo) (*tls.Certificate, error) {
					for _, cert := range remainingCertificates {
						return &cert, nil
					}
					return nil, fmt.Errorf("no certificate available")
				},
			}
			logging.System("Grace period ended. Old certificate removed.")
		}(oldTLSConfig.Certificates[0]) // Pass the old certificate for removal
	}
}

func certEqual(new, old tls.Certificate) bool {
	return string(old.Certificate[0]) == string(new.Certificate[0])
}

func watchCertificates(certPath, keyPath string, interval time.Duration) {
	var lastCertHash, lastKeyHash string

	logging.System("Watching certificate files for changes...")

	for {
		select {
		case <-time.After(interval):
			// Check if either file has changed
			certChanged := checkFileContentChange(certPath, &lastCertHash)
			keyChanged := checkFileContentChange(keyPath, &lastKeyHash)

			if certChanged || keyChanged {
				logging.System("Certificate content changed; reloading...")
				loadCertificates(certPath, keyPath)
			}
		}
	}
}

func checkFileContentChange(filePath string, lastHash *string) bool {
	// Compute the current hash of the file
	currentHash, err := hashFile(filePath)
	if err != nil {
		logging.Error(fmt.Sprintf("Failed to hash file %s: %v", filePath, err))
		return false
	}

	// Compare with the last hash
	if currentHash != *lastHash {
		logging.Debug(fmt.Sprintf("File content changed: %s", filePath))
		*lastHash = currentHash
		return true
	}

	return false
}

func hashFile(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	// Calculate the SHA256 hash of the file
	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}
