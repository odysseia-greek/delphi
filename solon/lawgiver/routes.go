package lawgiver

import (
	"github.com/gorilla/mux"
	"github.com/odysseia-greek/agora/plato/middleware"
	"github.com/odysseia-greek/delphi/solon/config"
)

// InitRoutes to start up a mux router and return the routes
func InitRoutes(config config.Config) *mux.Router {
	serveMux := mux.NewRouter()

	handler := SolonHandler{Config: &config}

	serveMux.HandleFunc("/solon/v1/ping", middleware.Adapt(handler.PingPong, middleware.ValidateRestMethod("GET"), middleware.SetCorsHeaders()))
	serveMux.HandleFunc("/solon/v1/health", middleware.Adapt(handler.Health, middleware.ValidateRestMethod("GET"), middleware.SetCorsHeaders()))
	serveMux.HandleFunc("/solon/v1/token", middleware.Adapt(handler.CreateOneTimeToken, middleware.ValidateRestMethod("GET"), middleware.LogRequestDetails(), middleware.SetCorsHeaders()))
	serveMux.HandleFunc("/solon/v1/register", middleware.Adapt(handler.RegisterService, middleware.ValidateRestMethod("POST"), middleware.LogRequestDetails(), middleware.SetCorsHeaders()))

	return serveMux
}
