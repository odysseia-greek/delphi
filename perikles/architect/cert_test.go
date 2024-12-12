package architect

import (
	"context"
	"github.com/odysseia-greek/agora/plato/certificates"
	kubernetes "github.com/odysseia-greek/agora/thales"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func TestCertCreation(t *testing.T) {
	ns := "test"
	secretName := "testsecret"
	hosts := []string{
		"perikles",
		"perikles.odysseia",
		"perikles.odysseia.svc",
		"perikles.odysseia.svc.cluster.local",
	}
	organizations := []string{"test"}
	validityCa := 3650

	cert, err := certificates.NewCertGeneratorClient(organizations, validityCa)
	assert.Nil(t, err)
	assert.NotNil(t, cert)
	err = cert.InitCa()
	assert.Nil(t, err)

	t.Run("SecretNewlyCreated", func(t *testing.T) {
		fakeKube := kubernetes.NewFakeKubeClient()
		assert.Nil(t, err)
		testConfig := Config{
			Kube:      fakeKube,
			Cert:      cert,
			Namespace: ns,
		}

		handler := PeriklesHandler{Config: &testConfig}
		err = handler.createCert(hosts, 1, secretName)
		assert.Nil(t, err)
	})

	t.Run("SecretAlreadyExists", func(t *testing.T) {
		fakeKube := kubernetes.NewFakeKubeClient()
		assert.Nil(t, err)
		testConfig := Config{
			Kube:      fakeKube,
			Cert:      cert,
			Namespace: ns,
		}

		data := map[string][]byte{
			"somesecret": []byte("verysecret"),
		}

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
			Type:      corev1.SecretTypeTLS,
		}
		fakeKube.CoreV1().Secrets(ns).Create(context.Background(), secret, metav1.CreateOptions{})

		handler := PeriklesHandler{Config: &testConfig}
		err = handler.createCert(hosts, 1, secretName)
		assert.Nil(t, err)
	})
}
