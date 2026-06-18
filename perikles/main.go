package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/odysseia-greek/agora/plato/logging"
	"github.com/odysseia-greek/delphi/perikles/architect"
)

const (
	crdReadySleep = 20 * time.Second
)

func main() {
	printBanner()

	logging.System("Bootstrapping Perikles configuration...")
	cfg, err := architect.CreateNewConfig()
	if err != nil {
		log.Fatal("death has found me")
	}

	logging.System(fmt.Sprintf("Namespace: %s", cfg.Namespace))
	logging.System(fmt.Sprintf("CRD name:   %s", cfg.CrdName))

	// Ensure CRD exists and mapping object exists
	if err := ensureCRDAndMapping(cfg); err != nil {
		log.Fatal(err.Error())
	}

	// Start background loops / watchers
	startControllers(cfg)

	logging.System("Perikles is running. Awaiting events.")
	select {} // keep process alive
}

func printBanner() {
	// https://patorjk.com/software/taag/#p=display&f=Crawford2&t=PERIKLES
	logging.System(`
 ____   ___  ____   ____  __  _  _        ___  _____
|    \ /  _]|    \ |    ||  |/ ]| |      /  _]/ ___/
|  o  )  [_ |  D  ) |  | |  ' / | |     /  [_(   \_ 
|   _/    _]|    /  |  | |    \ | |___ |    _]\__  |
|  | |   [_ |    \  |  | |     ||     ||   [_ /  \ |
|  | |     ||  .  \ |  | |  .  ||     ||     |\    |
|__| |_____||__|\_||____||__|\_||_____||_____| \___|
                                                    
`)
	logging.System(strings.Repeat("~", 37))
	logging.System("\"τόν γε σοφώτατον οὐχ ἁμαρτήσεται σύμβουλον ἀναμείνας χρόνον.\"")
	logging.System("\"he would yet do full well to wait for that wisest of all counsellors, Time.\"")
	logging.System(strings.Repeat("~", 37))
}

func ensureCRDAndMapping(cfg *architect.PeriklesHandler) error {
	logging.System("Ensuring Perikles CRD exists...")

	created, err := cfg.Mapping.CreateInCluster()
	if err != nil {
		return fmt.Errorf("failed to create/ensure CRD: %w", err)
	}

	if created {
		logging.System("CRD created. Waiting briefly for it to become established...")
		time.Sleep(crdReadySleep)
	} else {
		logging.System("CRD already present.")
	}

	logging.System("Ensuring mapping resource exists...")

	// If Get fails, we attempt to create the mapping resource (your previous logic).
	if _, err := cfg.Mapping.Get(cfg.CrdName); err == nil {
		logging.System("Mapping resource already present.")
		return nil
	}

	logging.System("Mapping resource not found; creating a fresh mapping...")
	mapping, err := cfg.Mapping.Parse(nil, cfg.CrdName, cfg.Namespace)
	if err != nil {
		return fmt.Errorf("failed to parse mapping template: %w", err)
	}

	createdCrd, err := cfg.Mapping.Create(mapping)
	if err != nil {
		return fmt.Errorf("failed to create mapping resource: %w", err)
	}

	logging.System(fmt.Sprintf("Created mapping: %s", createdCrd.Name))
	return nil
}

func startControllers(cfg *architect.PeriklesHandler) {
	logging.System("Starting control loops...")

	// These look like pure loops, so start them and log.
	logging.System("• Mapping update loop")
	go cfg.LoopForMappingUpdates()

	logging.System("• Pending update processor")
	go cfg.StartProcessingPendingUpdates()

	logging.System("• Stale network policy reconciler")
	go cfg.LoopForStaleNetworkPolicies()

	logging.System(fmt.Sprintf("• Dashboard on %s", cfg.DashboardAddr))
	go func() {
		if err := cfg.StartDashboard(); err != nil {
			logging.Error(fmt.Sprintf("Dashboard stopped: %v", err))
		}
	}()

	logging.System("• ConfigMap watcher")
	go func() {
		if err := cfg.WatchConfigMapChanges(); err != nil {
			logging.Error(fmt.Sprintf("ConfigMap watcher stopped: %v", err))
		}
	}()

	logging.System("• Workload watchers (deployments/pods)")
	go func() {
		if err := cfg.StartWatching(); err != nil {
			logging.Error(fmt.Sprintf("Workload watcher stopped: %v", err))
		}
	}()

	// Optional: a small “startup probe” log after a short delay to show we’re alive.
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = ctx // reserved for future: verify informers synced, etc.
		time.Sleep(250 * time.Millisecond)
		logging.System("All watchers started.")
	}()
}
