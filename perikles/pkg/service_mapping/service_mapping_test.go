package service_mapping

import (
	"testing"
	"time"

	"github.com/odysseia-greek/delphi/perikles/pkg/service_mapping/crd/v1alpha"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestServiceMappingClient(t *testing.T) {
	ns := "odysseia"
	updatedName := "fakemappingname"
	updatedApiVersion := "fakeversionafterupdate"
	testClient, err := NewFakeServiceMappingImpl()
	assert.Nil(t, err)

	t.Run("Get", func(t *testing.T) {
		sut, err := testClient.Get("sdfsdf")
		assert.Nil(t, err)
		assert.NotNil(t, sut)
	})

	t.Run("Create", func(t *testing.T) {
		sut, err := testClient.Create(nil)
		assert.Nil(t, err)
		assert.NotNil(t, sut)
	})

	t.Run("Update", func(t *testing.T) {
		mapping := v1alpha.Mapping{
			TypeMeta:   metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{},
			APIVersion: updatedApiVersion,
			Kind:       "",
			Spec:       v1alpha.Spec{},
		}
		mapping.Name = updatedName
		sut, err := testClient.Update(&mapping)
		assert.Nil(t, err)
		assert.NotNil(t, sut)
		assert.Equal(t, updatedName, sut.Name)
		assert.Equal(t, updatedApiVersion, sut.APIVersion)
	})

	t.Run("UpdateAndGet", func(t *testing.T) {
		mapping := v1alpha.Mapping{
			TypeMeta:   metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{},
			APIVersion: updatedApiVersion,
			Kind:       "",
			Spec:       v1alpha.Spec{},
		}
		mapping.Name = updatedName
		_, err = testClient.Update(&mapping)
		assert.Nil(t, err)
		sut, err := testClient.Get(updatedName)
		assert.Nil(t, err)
		assert.NotNil(t, sut)
		assert.Equal(t, updatedName, sut.Name)
		assert.Equal(t, updatedApiVersion, sut.APIVersion)
	})

	t.Run("ParseAndUpdate", func(t *testing.T) {
		validity := 9933345325
		kubeType := "TestKubeType"
		serviceName := "anewaddedservice"
		service := []v1alpha.Service{{
			Name:       serviceName,
			KubeType:   kubeType,
			SecretName: "",
			Namespace:  ns,
			Active:     false,
			Created:    time.Now().String(),
			Validity:   validity,
			Clients:    nil,
		},
		}
		mapping, err := testClient.Parse(service, updatedName, ns)
		assert.Nil(t, err)
		_, err = testClient.Update(mapping)
		assert.Nil(t, err)
		sut, err := testClient.Get(updatedName)
		assert.Nil(t, err)
		assert.NotNil(t, sut)
		assert.Equal(t, updatedName, sut.Name)
		assert.Equal(t, validity, sut.Spec.Services[0].Validity)
	})
}
