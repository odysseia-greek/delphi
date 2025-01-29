package architect

import (
	"github.com/odysseia-greek/agora/plato/logging"
	"time"
)

func (p *PeriklesHandler) loopForMappingUpdates() {
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

func (p *PeriklesHandler) startProcessingPendingUpdates() {
	ticker := time.NewTicker(p.PendingUpdateTimer)
	go func() {
		for range ticker.C {
			p.processPendingUpdates()
		}
	}()
}

func (p *PeriklesHandler) startReconciling() {
	ticker := time.NewTicker(p.ReconcileTimer)
	go func() {
		for range ticker.C {
			err := p.reconcileJob()
			if err != nil {
				logging.Error(err.Error())
			}
		}
	}()
}
