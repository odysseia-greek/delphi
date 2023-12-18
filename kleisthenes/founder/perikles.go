package founder

import (
	"context"
	"fmt"
	"github.com/odysseia-greek/agora/plato/certificates"
	"github.com/odysseia-greek/agora/plato/logging"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

const (
	TLSNAME = "tls"
)

func (k *KleisthenesHandler) perikles() error {
	logging.Debug("Setting up TLS for Perikles")
	secretName := "perikles-certs"

	validity := 3650

	orgName := []string{
		k.namespace,
	}

	hosts := []string{
		fmt.Sprintf("%s", k.periklesService),
		fmt.Sprintf("%s.%s", k.periklesService, k.namespace),
		fmt.Sprintf("%s.%s.svc", k.periklesService, k.namespace),
		fmt.Sprintf("%s.%s.svc.cluster.local", k.periklesService, k.namespace),
	}

	certClient, err := certificates.NewCertGeneratorClient(orgName, validity)
	if err != nil {
		return err
	}
	err = certClient.InitCa()
	if err != nil {
		return err
	}

	crt, key, _ := certClient.GenerateKeyAndCertSet(hosts, validity)

	certData := make(map[string][]byte)
	certData[fmt.Sprintf("%s.key", TLSNAME)] = key
	certData[fmt.Sprintf("%s.crt", TLSNAME)] = crt

	//caBundle := base64.StdEncoding.EncodeToString(crt)
	webhookName := "perikles-webhook"
	err = k.updateWebhookCA(webhookName, crt)
	if err != nil {
		return err
	}

	err = k.createSecret(secretName, certData, corev1.SecretTypeOpaque)
	if err != nil {
		return err
	}

	return err
}

func (k *KleisthenesHandler) updateWebhookCA(webhookName string, caBundle []byte) error {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	logging.Debug(fmt.Sprintf("updating webhook: %s", webhookName))

	webhook, err := k.Kube.AdmissionRegistrationV1().ValidatingWebhookConfigurations().Get(ctx, webhookName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	webhook.Webhooks[0].ClientConfig.CABundle = caBundle
	_, err = k.Kube.AdmissionRegistrationV1().ValidatingWebhookConfigurations().Update(ctx, webhook, metav1.UpdateOptions{})

	logging.Debug(fmt.Sprintf("updated webhook: %s", webhookName))
	return err
}

func (k *KleisthenesHandler) updatePeriklesAfterWebhook(deploymentName string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	deployment, err := k.Kube.AppsV1().Deployments(k.namespace).Get(ctx, deploymentName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	deployment.Spec.Template.Spec.Containers[0].Env = append(
		deployment.Spec.Template.Spec.Containers[0].Env,
		corev1.EnvVar{
			Name:  "trigger",
			Value: time.Now().Format(time.RFC3339Nano), // We just need a change to the spec to trigger a rolling update
		})

	updated, err := k.Kube.AppsV1().Deployments(k.namespace).Update(ctx, deployment, metav1.UpdateOptions{})

	logging.Debug(fmt.Sprintf("updated deployment after webhook: %s", updated.Name))
	return err
}
