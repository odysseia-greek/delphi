package architect

import (
	"context"
	"fmt"
	"github.com/odysseia-greek/agora/plato/logging"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
	"time"
)

// createCert generates TLS certificates and saves them as a Kubernetes Secret
func (p *PeriklesHandler) createCert(hosts []string, validityDays int, secretName string) error {
	tlsName := "tls"
	crt, key, err := p.Config.Cert.GenerateKeyAndCertSet(hosts, validityDays)
	if err != nil {
		return err
	}

	certData := make(map[string][]byte)
	certData[fmt.Sprintf("%s.key", tlsName)] = key
	certData[fmt.Sprintf("%s.crt", tlsName)] = crt
	certData[fmt.Sprintf("%s.pem", tlsName)] = p.Config.Cert.PemEncodedCa()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	secretExists := true
	_, err = p.Config.Kube.CoreV1().Secrets(p.Config.Namespace).Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			secretExists = false
		}
	}

	if secretExists {
		logging.Info(fmt.Sprintf("secret %s already exists", secretName))

		newAnnotation := make(map[string]string)
		newAnnotation[AnnotationUpdate] = time.Now().UTC().Format(timeFormat)
		immutable := false

		scr := corev1.Secret{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Secret",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:        secretName,
				Annotations: newAnnotation,
			},
			Immutable: &immutable,
			Data:      certData,
			Type:      corev1.SecretTypeTLS,
		}

		_, err := p.Config.Kube.CoreV1().Secrets(p.Config.Namespace).Update(context.Background(), &scr, metav1.UpdateOptions{})
		if err != nil {
			return err
		}
		return nil
	}

	logging.Info(fmt.Sprintf("secret %s does not exist", secretName))
	immutable := false
	scr := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: secretName,
		},
		Immutable: &immutable,
		Data:      certData,
		Type:      corev1.SecretTypeTLS,
	}
	_, err = p.Config.Kube.CoreV1().Secrets(p.Config.Namespace).Create(context.Background(), scr, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	return nil
}
