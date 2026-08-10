package architect

import (
	"context"
	"testing"
	"time"

	"github.com/odysseia-greek/agora/plato/certificates"
	"github.com/odysseia-greek/delphi/perikles/pkg/service_mapping"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func setupTestEnvironment() (*PeriklesHandler, string, string, string, string, string) {
	ns := "test"
	organizations := []string{"test"}
	validityCa := 3650

	cert, _ := certificates.NewCertGeneratorClient(organizations, validityCa)
	_ = cert.InitCa()

	fakeKube := newFakeKubeClient()
	mapping, _ := service_mapping.NewFakeServiceMappingImpl()

	handler := &PeriklesHandler{
		Kube:      fakeKube,
		Namespace: ns,
		CrdName:   "test",
		Mapping:   mapping,
	}

	serviceName := "test"
	existingServiceName := "fakedService"
	clientName := "testClient"
	secretName := "test-secret"
	kubeType := "Deployment"

	return handler, serviceName, existingServiceName, clientName, secretName, kubeType
}

func TestDurationDifference(t *testing.T) {
	valid := 10
	daysOfFuturePast := valid - 2*valid + 1

	t.Run("RedeployNeeded", func(t *testing.T) {
		created := time.Now().UTC().AddDate(0, 0, daysOfFuturePast).Format(timeFormat)
		redeploy, err := calculateTimeDifference(valid, created)
		assert.Nil(t, err)
		assert.True(t, redeploy)
	})
	t.Run("NoRedeployNeeded", func(t *testing.T) {
		created := time.Now().UTC().Format(timeFormat)
		redeploy, err := calculateTimeDifference(valid, created)
		assert.Nil(t, err)
		assert.False(t, redeploy)
	})
	t.Run("ErrorOnNoneFormattedTime", func(t *testing.T) {
		created := time.Now().UTC().String()
		redeploy, err := calculateTimeDifference(valid, created)
		assert.NotNil(t, err)
		assert.False(t, redeploy)
	})
}

func TestHostAndClientMapping(t *testing.T) {
	handler, serviceName, _, clientName, secretName, kubeType := setupTestEnvironment()

	t.Run("AddHostToMapping", func(t *testing.T) {
		update := MappingUpdate{
			HostName:   serviceName,
			SecretName: secretName,
			KubeType:   kubeType,
			Validity:   10,
		}
		err := handler.addHostToMapping(update)
		assert.Nil(t, err)
	})

	t.Run("AddClientToNewService", func(t *testing.T) {
		err := handler.addClientToMapping(MappingUpdate{
			HostName:   serviceName,
			ClientName: clientName,
			KubeType:   kubeType,
		})
		assert.Nil(t, err)
	})
}

func TestCheckMappingForUpdates(t *testing.T) {
	handler, serviceName, _, _, secretName, kubeType := setupTestEnvironment()

	t.Run("NoRedeployNeeded", func(t *testing.T) {
		err := handler.checkMappingForUpdates()
		assert.Nil(t, err)
	})

	t.Run("RedeployNeeded", func(t *testing.T) {
		update := MappingUpdate{
			HostName:   serviceName,
			SecretName: secretName,
			KubeType:   kubeType,
			Validity:   10,
		}
		err := handler.addHostToMapping(update)
		assert.Nil(t, err)

		deploy := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: serviceName, Namespace: handler.Namespace}}
		_, err = handler.Kube.AppsV1().Deployments(handler.Namespace).Create(context.Background(), deploy, metav1.CreateOptions{})
		assert.Nil(t, err)

		err = handler.checkMappingForUpdates()
		assert.Nil(t, err)
	})
}
