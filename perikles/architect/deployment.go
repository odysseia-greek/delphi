package architect

import (
	"context"
	"fmt"
	"github.com/odysseia-greek/agora/plato/logging"
	"k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strconv"
	"strings"
	"time"
)

const (
	AnnotationUpdate     = "perikles/updated"
	AnnotationValidity   = "perikles/validity"
	AnnotationHost       = "perikles/hostname"
	AnnotationAccesses   = "perikles/accesses"
	AnnotationHostSecret = "perikles/hostsecret"
)

func (p *PeriklesHandler) checkForAnnotations(deployment v1.Deployment) error {
	for key, value := range deployment.Spec.Template.Annotations {
		if key == AnnotationHost {
			err := p.hostFlow(deployment)
			if err != nil {
				return err
			}
		}

		if key == AnnotationAccesses {
			err := p.clientFlow(value, deployment.Name, deployment.Kind)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (p *PeriklesHandler) hostFlow(deployment v1.Deployment) error {
	var validity int
	var hostName string
	var secretName string

	for key, value := range deployment.Spec.Template.Annotations {
		logging.Info("looking through annotation")

		if key == AnnotationValidity {
			validity, _ = strconv.Atoi(value)
			logging.Info(fmt.Sprintf("validity set to %v", validity))
		}

		if key == AnnotationHost {
			hostName = value
			logging.Info(fmt.Sprintf("host set to %s", hostName))
		}

		if key == AnnotationHostSecret {
			secretName = value
			logging.Info(fmt.Sprintf("secretName set to %s", secretName))
		}
	}

	if secretName == "" {
		for _, volume := range deployment.Spec.Template.Spec.Volumes {
			if volume.Secret != nil {
				if strings.Contains(volume.Secret.SecretName, hostName) || volume.Name == hostName {
					secretName = volume.Secret.SecretName
					logging.Info(fmt.Sprintf("secretName set to %s", secretName))
				}
			}
		}
	}

	orgName := deployment.Namespace

	hosts := []string{
		fmt.Sprintf("%s", hostName),
		fmt.Sprintf("%s.%s", hostName, orgName),
		fmt.Sprintf("%s.%s.svc", hostName, orgName),
		fmt.Sprintf("%s.%s.svc.cluster.local", hostName, orgName),
	}

	logging.Debug("creating certs")
	err := p.createCert(hosts, validity, secretName)
	if err != nil {
		return err
	}

	logging.Debug("created certs")
	_, err = p.addHostToMapping(hostName, secretName, deployment.Kind, validity)
	if err != nil {
		return err
	}

	//TODO: fix the restart since its stuck in a loop
	//glg.Debug("restarting deployment")
	//err = p.restartKubeResource(deployment.Namespace, deployment.Name, deployment.Kind)
	//if err != nil {
	//	return err
	//}

	return nil
}

func (p *PeriklesHandler) clientFlow(accesses, name, kubeType string) error {
	hosts := strings.Split(accesses, ";")

	for _, host := range hosts {
		_, err := p.addClientToMapping(host, name, kubeType)
		if err != nil {
			return err
		}
	}

	return nil
}

func (p *PeriklesHandler) restartDeployment(ns, deploymentName string) error {
	newAnnotation := make(map[string]string)
	newAnnotation[AnnotationUpdate] = time.Now().UTC().Format(timeFormat)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	deployment, err := p.Config.Kube.AppsV1().Deployments(p.Config.Namespace).Get(ctx, deploymentName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	for key, value := range newAnnotation {
		deployment.Spec.Template.Annotations[key] = value
	}

	_, err = p.Config.Kube.AppsV1().Deployments(p.Config.Namespace).Update(ctx, deployment, metav1.UpdateOptions{})

	return err
}

func (p *PeriklesHandler) restartKubeResource(ns, name, kubeType string) error {
	switch kubeType {
	case "Deployment":
		err := p.restartDeployment(ns, name)
		return err
	}

	return nil
}
