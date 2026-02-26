package architect

import (
	"time"

	"github.com/odysseia-greek/agora/plato/logging"
)

func (p *PeriklesHandler) LoopForMappingUpdates() {
	ticker := time.NewTicker(p.TLSCheckTimer)
	for {
		select {
		case <-ticker.C:
			err := p.checkMappingForUpdates()
			if err != nil {
				logging.Error(err.Error())
			}
		}
	}
}

func (p *PeriklesHandler) StartProcessingPendingUpdates() {
	ticker := time.NewTicker(p.PendingUpdateTimer)
	go func() {
		for range ticker.C {
			p.processPendingUpdates()
		}
	}()
}
