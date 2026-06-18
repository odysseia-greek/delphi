package architect

import (
	"fmt"
	"strconv"
	"strings"

	v1 "k8s.io/api/apps/v1"
)

func (p *PeriklesHandler) checkForAnnotations(deployment *v1.Deployment) error {
	annotations := deployment.Spec.Template.Annotations
	kubeType := deployment.Kind
	if kubeType == "" {
		kubeType = "Deployment"
	}

	for _, hostName := range splitAnnotationList(annotations[AnnotationAccesses]) {
		p.addClientToPendingUpdates(hostName, deployment.Name, kubeType, "", deployment.Namespace, 0, false)
	}

	hostName := strings.TrimSpace(annotations[AnnotationHost])
	if hostName == "" {
		return nil
	}

	validity := 0
	if rawValidity := strings.TrimSpace(annotations[AnnotationValidity]); rawValidity != "" {
		parsedValidity, err := strconv.Atoi(rawValidity)
		if err != nil {
			return fmt.Errorf("invalid %s annotation %q on deployment %s/%s: %w",
				AnnotationValidity, rawValidity, deployment.Namespace, deployment.Name, err)
		}
		validity = parsedValidity
	}

	secretName := findHostSecretName(deployment, hostName)
	p.addClientToPendingUpdates(hostName, "", kubeType, secretName, deployment.Namespace, validity, true)
	return nil
}

func splitAnnotationList(value string) []string {
	parts := strings.Split(value, ";")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		if value := strings.TrimSpace(part); value != "" {
			values = append(values, value)
		}
	}
	return values
}

func findHostSecretName(deployment *v1.Deployment, hostName string) string {
	for _, volume := range deployment.Spec.Template.Spec.Volumes {
		if volume.Secret != nil && strings.Contains(volume.Secret.SecretName, hostName) {
			return volume.Secret.SecretName
		}
	}
	return fmt.Sprintf("%s-tls-certs", hostName)
}
