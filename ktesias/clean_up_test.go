package ktesias

import (
	"context"
	"fmt"
	"github.com/odysseia-greek/agora/plato/logging"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

func (l *OdysseiaFixture) cleanupResources() {
	cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resources := getResourcesFromContext(l.ctx)
	for _, resource := range resources {
		var err error
		switch resource.Kind {
		case "Deployment":
			err = l.Kube.AppsV1().Deployments(l.Namespace).Delete(cleanupCtx, resource.Name, metav1.DeleteOptions{})
		default:
			logging.Warn(fmt.Sprintf("Unsupported resource type: %s", resource.Kind))
		}
		if err != nil {
			logging.Error(fmt.Sprintf("Failed to delete %s %s: %v", resource.Kind, resource.Name, err))
		}
	}
}
