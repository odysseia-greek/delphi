package ktesias

import (
	"context"
	"fmt"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
	"time"
)

func (l *OdysseiaFixture) aSecretShouldBeCreatedForTlsCertsForHost(hostname string) error {
	secretName := fmt.Sprintf("%s-tls-certs", hostname)
	requiredKeys := []string{"tls.key", "tls.pem", "tls.crt"}

	return retryWithTimeout(10*time.Second, time.Second, func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		secret, err := l.Kube.CoreV1().Secrets(l.Namespace).Get(ctx, secretName, metav1.GetOptions{})
		if err != nil {
			return err
		}

		for _, key := range requiredKeys {
			if _, exists := secret.Data[key]; !exists {
				return fmt.Errorf("missing required key %s in secret %s", key, secretName)
			}
		}

		return nil
	})
}

func (l *OdysseiaFixture) ciliumNetWorkPoliciesShouldExistForRoleFromHost(role, hostname string) error {
	name := fmt.Sprintf("restrict-elasticsearch-access-%s", hostname)

	return retryWithTimeout(10*time.Second, time.Second, func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		cnp, err := l.CiliumClient.CiliumV2().CiliumNetworkPolicies(l.Namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("failed to retrieve CiliumNetworkPolicy %s: %w", name, err)
		}

		for key, value := range cnp.Spec.EndpointSelector.LabelSelector.MatchLabels {
			if strings.Contains(key, "elasticsearch.k8s.elastic.co/cluster-name") {
				if value != "aristoteles" {
					return fmt.Errorf("unexpected app label in EndpointSelector: got %s, want %s", value, "aristoteles")
				}
			}
		}

		if role == "api" {
			for _, ingressRule := range cnp.Spec.Ingress {
				if len(ingressRule.ToPorts) > 0 {
					for _, toPort := range ingressRule.ToPorts {
						if len(toPort.Rules.HTTP) > 0 {
							for _, httpRule := range toPort.Rules.HTTP {
								if httpRule.Method == "^GET$" {
									if !(httpRule.Path == "^/$") {
										return fmt.Errorf("health endpoint HTTP rule not found")
									}
									continue
								}
								if !(httpRule.Method == "^POST$" && (strings.Contains(httpRule.Path, "_search") || strings.Contains(httpRule.Path, "scroll"))) {
									return fmt.Errorf("invalid HTTP rule: %+v", httpRule)
								}
							}
						} else {
							return fmt.Errorf("L7 rules are missing or invalid in ToPorts configuration")
						}
					}
				}
			}
		}

		return nil
	})
}

func (l *OdysseiaFixture) aCiliumNetWorkPolicyShouldExistForAccessFromTheDeploymentToTheHost(deploymentName, hostName string) error {
	name := fmt.Sprintf("allow-%s-access-%s", deploymentName, hostName)

	return retryWithTimeout(10*time.Second, time.Second, func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		cnp, err := l.CiliumClient.CiliumV2().CiliumNetworkPolicies(l.Namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("failed to retrieve CiliumNetworkPolicy %s: %w", name, err)
		}

		for key, value := range cnp.Spec.EndpointSelector.LabelSelector.MatchLabels {
			if key == "app" {
				if value != hostName {
					return fmt.Errorf("unexpected app label in EndpointSelector: got %s, want %s", value, hostName)
				}
			}
		}
		for _, ingressRule := range cnp.Spec.Ingress {
			for _, endpoint := range ingressRule.FromEndpoints {
				for key, value := range endpoint.MatchLabels {
					if key == "app" {
						if value != deploymentName {
							return fmt.Errorf("unexpected app label in EndpointSelector: got %s, want %s", value, deploymentName)
						}
					}
				}
			}
		}

		return nil
	})
}

func (l *OdysseiaFixture) theCreatedResourceIsCheckedAfterAWait(hostname string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	deployment, err := l.Kube.AppsV1().Deployments(l.Namespace).Get(ctx, hostname, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to create deployment: %v", err)
	}

	if deployment == nil {
		return fmt.Errorf("deployment does not exist")
	}

	return nil
}

func (l *OdysseiaFixture) aDeploymentIsCreatedWithRoleAccessHostAndBeingAClientOf(role, access, hostname, client string) error {
	l.ctx = addResourceToContext(l.ctx, Resource{Kind: "Deployment", Name: hostname})

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: hostname,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": hostname},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": hostname},
					Annotations: map[string]string{
						"odysseia-greek/role":   role,
						"odysseia-greek/access": access,
						"perikles/accesses":     client,
						"perikles/hostname":     hostname,
						"perikles/validity":     "10",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "nginx",
							Image: "nginx:latest",
						},
					},
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := l.Kube.AppsV1().Deployments(l.Namespace).Create(ctx, deployment, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create deployment: %v", err)
	}

	return nil
}
