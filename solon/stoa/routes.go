package stoa

import (
	"net/http"

	"github.com/odysseia-greek/agora/plato/middleware"
	"github.com/odysseia-greek/attike/aristophanes/comedy"
	"github.com/odysseia-greek/delphi/solon/lawgiver"
)

// InitRoutes creates Solon's HTTP routes.
func InitRoutes(solonHandler *lawgiver.SolonHandler) *http.ServeMux {
	serveMux := http.NewServeMux()

	serveMux.HandleFunc("/solon/v1/health", middleware.Adapt(solonHandler.Health, middleware.ValidateRestMethod("GET")))
	serveMux.HandleFunc("/solon/v1/token", middleware.Adapt(solonHandler.CreateOneTimeToken, middleware.ValidateRestMethod("GET"), middleware.Adapter(comedy.TraceWithHopStop(solonHandler.Streamer))))
	serveMux.HandleFunc("/solon/v1/register", middleware.Adapt(solonHandler.RegisterService, middleware.ValidateRestMethod("POST"), middleware.LogRequestDetails()))

	return serveMux
}
