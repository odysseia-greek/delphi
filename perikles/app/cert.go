package app

import (
	"context"
	"fmt"
	"github.com/odysseia-greek/agora/plato/logging"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

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

	secret, _ := p.Config.Kube.CoreV1().Secrets(p.Config.Namespace).Get(context.Background(), secretName, metav1.GetOptions{})

	if secret == nil {
		logging.Info(fmt.Sprintf("secret %s does not exist", secretName))
		immutable := false
		secret := &corev1.Secret{
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
		_, err = p.Config.Kube.CoreV1().Secrets(p.Config.Namespace).Create(context.Background(), secret, metav1.CreateOptions{})
		if err != nil {
			return err
		}
	} else {
		logging.Info(fmt.Sprintf("secret %s already exists", secret.Name))

		newAnnotation := make(map[string]string)
		newAnnotation[AnnotationUpdate] = time.Now().UTC().Format(timeFormat)
		immutable := false

		secret := corev1.Secret{
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

		_, err := p.Config.Kube.CoreV1().Secrets(p.Config.Namespace).Update(context.Background(), &secret, metav1.UpdateOptions{})
		if err != nil {
			return err
		}
		return nil
	}

	return nil
}
