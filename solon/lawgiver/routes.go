package lawgiver

import (
	"github.com/gorilla/mux"
	"github.com/odysseia-greek/agora/plato/middleware"
	"github.com/odysseia-greek/attike/aristophanes/comedy"
)

// InitRoutes to start up a mux router and return the routes
func InitRoutes(solonHandler *SolonHandler) *mux.Router {
	serveMux := mux.NewRouter()

	serveMux.HandleFunc("/solon/v1/ping", middleware.Adapt(solonHandler.PingPong, middleware.ValidateRestMethod("GET")))
	serveMux.HandleFunc("/solon/v1/health", middleware.Adapt(solonHandler.Health, middleware.ValidateRestMethod("GET")))
	serveMux.HandleFunc("/solon/v1/token", middleware.Adapt(solonHandler.CreateOneTimeToken, middleware.ValidateRestMethod("GET"), middleware.Adapter(comedy.TraceWithLogAndSpan(solonHandler.Streamer))))
	serveMux.HandleFunc("/solon/v1/register", middleware.Adapt(solonHandler.RegisterService, middleware.ValidateRestMethod("POST"), middleware.LogRequestDetails()))

	return serveMux
}
