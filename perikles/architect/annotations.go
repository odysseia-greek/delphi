package architect

import (
	"fmt"
	v1 "k8s.io/api/apps/v1"
	"strconv"
	"strings"
)

func (p *PeriklesHandler) checkForAnnotations(deployment *v1.Deployment) error {
	// Initialize variables for host-specific logic
	var validity int
	var hostName, secretName string

	// Parse deployment annotations
	for key, value := range deployment.Spec.Template.Annotations {
		switch key {
		case AnnotationValidity:
			validity, _ = strconv.Atoi(value)
		case AnnotationHost:
			hostName = value
		case AnnotationHostSecret:
			secretName = value
		case AnnotationAccesses:
			// Split the accesses list and queue each client relationship
			accessList := strings.Split(value, ";")
			for _, client := range accessList {
				p.addClientToPendingUpdates(client, deployment.Name, deployment.Kind, "", 0, false)
			}
		}
	}

	// Infer the secret name if it wasn't provided explicitly
	if secretName == "" {
		for _, volume := range deployment.Spec.Template.Spec.Volumes {
			if volume.Secret != nil && strings.Contains(volume.Secret.SecretName, hostName) {
				secretName = volume.Secret.SecretName
			}
		}
	}

	// the secretname might still be empty because the deployment has no mount this is not expected but a secret can still be created
	if secretName == "" {
		secretName = fmt.Sprintf("%s-tls-certs", hostName)
	}

	// If it's a valid host, add it to the pending updates
	if hostName != "" && secretName != "" {
		// Generate hostnames for certificate creation
		orgName := deployment.Namespace
		hosts := []string{
			fmt.Sprintf("%s", hostName),
			fmt.Sprintf("%s.%s", hostName, orgName),
			fmt.Sprintf("%s.%s.svc", hostName, orgName),
			fmt.Sprintf("%s.%s.svc.cluster.local", hostName, orgName),
		}

		// Create the TLS certificate
		err := p.createCert(hosts, validity, secretName)
		if err != nil {
			return err
		}

		// Queue the host update
		p.addClientToPendingUpdates(hostName, "", deployment.Kind, secretName, validity, true)
	}

	return nil
}
