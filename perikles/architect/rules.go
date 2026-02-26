package architect

import (
	"context"
	"fmt"
	"github.com/odysseia-greek/agora/plato/logging"
	"gopkg.in/yaml.v3"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (p *PeriklesHandler) WatchConfigMapChanges() error {
	configMapClient := p.Kube.CoreV1().ConfigMaps(p.Namespace)

	// Watch for changes in the ConfigMap
	watch, err := configMapClient.Watch(context.Background(), metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to watch config map: %v", err)
	}

	go func() {
		for event := range watch.ResultChan() {
			if event.Type == "MODIFIED" || event.Type == "ADDED" {
				// Get the ConfigMap object
				configMap, ok := event.Object.(*v1.ConfigMap)
				if !ok {
					logging.Error("Received non-ConfigMap event, ignoring")
					continue
				}

				// Check if this is the ConfigMap we care about (e.g., by name)
				if configMap.Name == p.ConfigMapName {
					logging.Debug(fmt.Sprintf("ConfigMap %s changed: %s", configMap.Name, event.Type))

					err := p.setCnpRules(configMap.Data)
					if err != nil {
						logging.Error(fmt.Sprintf("Failed to handle config map change: %v", err))
					}
				}
			}
		}
	}()

	select {}
}

func (p *PeriklesHandler) setCnpRules(configMap map[string]string) error {
	// Loop through each role in the configMap
	var mappings []CnpRuleSet
	for roleName, roleData := range configMap {
		// Unmarshal the role data (YAML) into a Role struct
		var role CnpElasticMapping
		err := yaml.Unmarshal([]byte(roleData), &role)
		if err != nil {
			return fmt.Errorf("failed to unmarshal config map data for role %s: %v", roleName, err)
		}

		mappings = append(mappings, CnpRuleSet{
			RoleName: roleName,
			CnpRules: role.CnpRules,
		})
	}

	p.RuleSet = mappings
	return nil
}
