package architect

import (
	"context"
	"github.com/odysseia-greek/agora/plato/certificates"
	kubernetes "github.com/odysseia-greek/agora/thales"
	"github.com/odysseia-greek/agora/thales/odysseia"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
	"time"
)

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

func TestSettingOfMappings(t *testing.T) {
	ns := "test"
	organizations := []string{"test"}
	validityCa := 3650
	cert, err := certificates.NewCertGeneratorClient(organizations, validityCa)
	assert.Nil(t, err)
	assert.NotNil(t, cert)
	err = cert.InitCa()
	assert.Nil(t, err)
	fakeKube := kubernetes.NewFakeKubeClient()
	mapping, err := odysseia.NewFakeServiceMappingImpl()
	assert.Nil(t, err)
	testConfig := Config{
		Kube:      fakeKube,
		Cert:      cert,
		Namespace: ns,
		CrdName:   "test",
		Mapping:   mapping,
	}
	handler := PeriklesHandler{Config: &testConfig}
	serviceName := "test"
	existingServiceName := "fakedService"
	clientName := "testClient"
	secretName := "test-secret"
	kubeType := "Deployment"
	crdName := "testCrd"

	t.Run("AddHostToMapping", func(t *testing.T) {
		sut, err := handler.addHostToMapping(serviceName, secretName, kubeType, 10)
		assert.Nil(t, err)
		assert.Equal(t, sut.Name, crdName)
		assert.Equal(t, sut.Spec.Services[0].Name, existingServiceName)
	})

	t.Run("UpdateExistingMapping", func(t *testing.T) {
		sut, err := handler.addHostToMapping(existingServiceName, secretName, kubeType, 10)
		assert.Nil(t, err)
		assert.Equal(t, sut.Name, crdName)
		assert.Equal(t, existingServiceName, sut.Spec.Services[0].Name)
		assert.Equal(t, secretName, sut.Spec.Services[0].SecretName)
	})

	t.Run("AddClientToNewServiceAndToMapping", func(t *testing.T) {
		sut, err := handler.addClientToMapping(serviceName, clientName, kubeType)
		assert.Nil(t, err)
		assert.True(t, len(sut.Spec.Services) >= 2)

		nameFound := false
		for _, service := range sut.Spec.Services {
			if service.Name == serviceName {
				for _, client := range service.Clients {
					if client.Name == clientName {
						nameFound = true
					}
				}
			}
		}
		assert.True(t, nameFound)
	})

	t.Run("AddClientToServiceAndToMapping", func(t *testing.T) {
		fk := kubernetes.NewFakeKubeClient()
		handler.Config.Kube = fk
		sut, err := handler.addClientToMapping(existingServiceName, clientName, kubeType)
		assert.Nil(t, err)
		assert.True(t, len(sut.Spec.Services[0].Clients) >= 2)
		assert.Equal(t, clientName, sut.Spec.Services[0].Clients[1].Name)
	})
}

func TestCheckMappingForUpdates(t *testing.T) {
	ns := "test"
	organizations := []string{"test"}
	validityCa := 3650
	cert, err := certificates.NewCertGeneratorClient(organizations, validityCa)
	assert.Nil(t, err)
	assert.NotNil(t, cert)
	err = cert.InitCa()
	assert.Nil(t, err)
	fakeKube := kubernetes.NewFakeKubeClient()
	mapping, err := odysseia.NewFakeServiceMappingImpl()
	assert.Nil(t, err)
	testConfig := Config{
		Kube:      fakeKube,
		Cert:      cert,
		Namespace: ns,
		Mapping:   mapping,
		CrdName:   "test",
	}
	handler := PeriklesHandler{Config: &testConfig}
	serviceName := "serviceNameForATestObjectThatCanBeRestarted"
	secretName := "test-secret"
	kubeType := "Deployment"

	t.Run("NoRedeployNeeded", func(t *testing.T) {
		sut := handler.checkMappingForUpdates()
		assert.Nil(t, sut)
	})

	t.Run("RedeployNeeded", func(t *testing.T) {
		_, err := handler.addHostToMapping(serviceName, secretName, kubeType, 1)
		assert.Nil(t, err)

		deploy := kubernetes.TestDeploymentObject(serviceName, ns)
		_, err = fakeKube.AppsV1().Deployments(ns).Create(context.Background(), deploy, metav1.CreateOptions{})
		assert.Nil(t, err)
		sut := handler.checkMappingForUpdates()
		assert.Nil(t, sut)
	})
}
