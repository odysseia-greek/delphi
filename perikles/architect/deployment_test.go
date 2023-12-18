package architect

import (
	"github.com/odysseia-greek/agora/plato/certificates"
	kubernetes "github.com/odysseia-greek/agora/thales"
	"github.com/odysseia-greek/agora/thales/odysseia"
	"github.com/odysseia-greek/delphi/perikles/config"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

func TestAnnotations(t *testing.T) {
	ns := "test"
	organizations := []string{"test"}
	validityCa := 3650
	cert, err := certificates.NewCertGeneratorClient(organizations, validityCa)
	assert.Nil(t, err)
	assert.NotNil(t, cert)
	err = cert.InitCa()
	assert.Nil(t, err)
	fakeKube := kubernetes.NewFakeKubeClient()
	fakeMapping, err := odysseia.NewFakeServiceMappingImpl()
	assert.Nil(t, err)
	crdName := "testCrd"
	testConfig := config.Config{
		Kube:      fakeKube,
		Mapping:   fakeMapping,
		Cert:      cert,
		Namespace: ns,
		CrdName:   crdName,
	}
	handler := PeriklesHandler{Config: &testConfig}
	deploymentName := "periklesDeployment"
	volumeName := "periklesVolume"
	host := "perikles"
	validity := "10"
	secretName := "periklesVolumeSecret"

	t.Run("HostOnly", func(t *testing.T) {
		annotations := map[string]string{
			AnnotationHost:     host,
			AnnotationValidity: validity,
		}
		deployment := kubernetes.TestAnnotatedDeploymentObject(deploymentName, ns, annotations)
		err := handler.checkForAnnotations(*deployment)
		assert.Nil(t, err)
		sut, err := testConfig.Mapping.Get("asfasf")
		assert.Nil(t, err)
		found := false
		for _, service := range sut.Spec.Services {
			if service.Name == host {
				assert.Equal(t, "", service.SecretName)
				found = true
			}
		}
		assert.True(t, found)
	})

	t.Run("HostOnlyWithSecretFromVolume", func(t *testing.T) {
		annotations := map[string]string{
			AnnotationHost:     host,
			AnnotationValidity: validity,
		}
		deployment := kubernetes.TestAnnotatedDeploymentObject(deploymentName, ns, annotations)
		volume := kubernetes.TestPodSpecVolume(volumeName, secretName)
		deployment.Spec.Template.Spec.Volumes = volume
		err := handler.checkForAnnotations(*deployment)
		assert.Nil(t, err)
		sut, err := testConfig.Mapping.Get("asfasf")
		assert.Nil(t, err)
		found := false
		for _, service := range sut.Spec.Services {
			if service.Name == host {
				assert.Equal(t, secretName, service.SecretName)
				found = true
			}
		}
		assert.True(t, found)
	})

	t.Run("HostOnlyWithSecretFromAnnotation", func(t *testing.T) {
		hostSecret := "superSecret"
		annotations := map[string]string{
			AnnotationHost:       host,
			AnnotationValidity:   validity,
			AnnotationHostSecret: hostSecret,
		}
		deployment := kubernetes.TestAnnotatedDeploymentObject(deploymentName, ns, annotations)
		err := handler.checkForAnnotations(*deployment)
		assert.Nil(t, err)
		sut, err := testConfig.Mapping.Get("asfasf")
		assert.Nil(t, err)
		found := false
		for _, service := range sut.Spec.Services {
			if service.Name == host {
				assert.Equal(t, hostSecret, service.SecretName)
				found = true
			}
		}
		assert.True(t, found)
	})

	t.Run("HostOnlyWithSecretFromAnnotation", func(t *testing.T) {
		hostSecret := "superSecret"
		annotations := map[string]string{
			AnnotationHost:       host,
			AnnotationValidity:   validity,
			AnnotationHostSecret: hostSecret,
		}
		deployment := kubernetes.TestAnnotatedDeploymentObject(deploymentName, ns, annotations)
		err := handler.checkForAnnotations(*deployment)
		assert.Nil(t, err)
		sut, err := testConfig.Mapping.Get("asfasf")
		assert.Nil(t, err)
		found := false
		for _, service := range sut.Spec.Services {
			if service.Name == host {
				assert.Equal(t, hostSecret, service.SecretName)
				found = true
			}
		}
		assert.True(t, found)
	})

	t.Run("ClientOnlyWithNonExistingService", func(t *testing.T) {
		client := "archimedes;plato"
		annotations := map[string]string{
			AnnotationAccesses: client,
		}
		deployment := kubernetes.TestAnnotatedDeploymentObject(deploymentName, ns, annotations)
		err := handler.checkForAnnotations(*deployment)
		assert.Nil(t, err)
		sut, err := testConfig.Mapping.Get("asfasf")
		assert.Nil(t, err)

		clients := strings.Split(client, ";")
		for _, service := range sut.Spec.Services {
			for _, c := range clients {
				if service.Name == c {
					assert.False(t, service.Active)
					assert.Equal(t, 1, len(service.Clients))
				}
			}
		}
	})
}
