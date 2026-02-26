package architect

import (
	"strings"

	v1 "k8s.io/api/apps/v1"
)

func (p *PeriklesHandler) checkForAnnotations(deployment *v1.Deployment) error {
	// Parse deployment annotations
	for key, value := range deployment.Spec.Template.Annotations {
		switch key {
		case AnnotationAccesses:
			// Split the accesses list and queue each client relationship
			accessList := strings.Split(value, ";")
			for _, client := range accessList {
				p.addClientToPendingUpdates(client, deployment.Name, deployment.Kind, "", deployment.Namespace, 0, false)
			}
		}
	}

	return nil
}
