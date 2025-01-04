package architect

import (
	"github.com/gorilla/mux"
	"github.com/odysseia-greek/agora/plato/middleware"
)

// InitRoutes to start up a mux router and return the routes
func InitRoutes(periklesHandler *PeriklesHandler) *mux.Router {
	serveMux := mux.NewRouter()

	serveMux.HandleFunc("/perikles/v1/ping", middleware.Adapt(periklesHandler.pingPong, middleware.ValidateRestMethod("GET"), middleware.SetCorsHeaders()))
	serveMux.HandleFunc("/perikles/v1/validate", middleware.Adapt(periklesHandler.validate, middleware.ValidateRestMethod("POST"), middleware.LogRequestDetails(), middleware.SetCorsHeaders()))

	go periklesHandler.loopForMappingUpdates()
	go periklesHandler.startProcessingPendingUpdates()
	go periklesHandler.startReconciling()

	return serveMux
}
