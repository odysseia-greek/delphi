package architect

import (
	"context"
	"fmt"
	"github.com/odysseia-greek/agora/plato/logging"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

const (
	IgnoreInGitOps       = "gitops.ignore"
	AnnotationUpdate     = "perikles/updated"
	AnnotationValidity   = "perikles/validity"
	AnnotationHost       = "perikles/hostname"
	AnnotationAccesses   = "perikles/accesses"
	AnnotationHostSecret = "perikles/hostsecret"
)

func (p *PeriklesHandler) restartDeployment(ns, deploymentName string) error {
	newAnnotation := make(map[string]string)
	newAnnotation[AnnotationUpdate] = time.Now().UTC().Format(timeFormat)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	deployment, err := p.Config.Kube.AppsV1().Deployments(ns).Get(ctx, deploymentName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	for key, value := range newAnnotation {
		deployment.Spec.Template.Annotations[key] = value
	}

	deploy, err := p.Config.Kube.AppsV1().Deployments(ns).Update(ctx, deployment, metav1.UpdateOptions{})
	if deploy != nil {
		logging.Info(fmt.Sprintf("restarting deployment %s in ns %s", deploy.Name, deploy.Namespace))
	}

	return err
}
