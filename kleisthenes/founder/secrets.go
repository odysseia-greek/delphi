package founder

import (
	"context"
	"fmt"
	"github.com/odysseia-greek/agora/plato/logging"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
	"time"
)

func (k *KleisthenesHandler) createSecret(secretName string, data map[string][]byte, secretType corev1.SecretType) error {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	secretExists := true
	_, err := k.Kube.CoreV1().Secrets(k.namespace).Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			secretExists = false
		}
	}

	if secretExists {
		err = k.Kube.CoreV1().Secrets(k.namespace).Delete(context.Background(), secretName, metav1.DeleteOptions{})
		if err != nil {
			return err
		}
	}

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
		Data:      data,
		Type:      secretType,
	}
	creation, err := k.Kube.CoreV1().Secrets(k.namespace).Create(ctx, secret, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	logging.Debug(fmt.Sprintf("created new secret: %s", creation.Name))
	return nil
}
