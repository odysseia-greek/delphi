package lawgiver

import (
	"fmt"
	"github.com/gorilla/mux"
	"github.com/odysseia-greek/agora/plato/logging"
	"github.com/odysseia-greek/agora/plato/middleware"
	"github.com/odysseia-greek/attike/aristophanes/comedy"
	"time"
)

// InitRoutes to start up a mux router and return the routes
func InitRoutes(solonHandler *SolonHandler, ticker time.Duration) *mux.Router {
	serveMux := mux.NewRouter()

	serveMux.HandleFunc("/solon/v1/health", middleware.Adapt(solonHandler.Health, middleware.ValidateRestMethod("GET")))
	serveMux.HandleFunc("/solon/v1/token", middleware.Adapt(solonHandler.CreateOneTimeToken, middleware.ValidateRestMethod("GET"), middleware.Adapter(comedy.TraceWithLogAndSpan(solonHandler.Streamer))))
	serveMux.HandleFunc("/solon/v1/register", middleware.Adapt(solonHandler.RegisterService, middleware.ValidateRestMethod("POST"), middleware.LogRequestDetails()))

	go func() {
		if err := solonHandler.StartCleanupService(ticker); err != nil {
			logging.Error(fmt.Sprintf("cleanup service failed with an error: %s", err.Error()))
		}
	}()

	return serveMux
}
