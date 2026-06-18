package architect

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestCheckForAnnotationsQueuesHostAndClients(t *testing.T) {
	handler := &PeriklesHandler{
		PendingUpdates: make(map[string][]MappingUpdate),
	}
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "solon",
			Namespace: "delphi",
		},
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						AnnotationHost:     "solon",
						AnnotationValidity: "365",
						AnnotationAccesses: " aristarchos ; herodotos; ",
					},
				},
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{
							Name: "solon-certs",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{SecretName: "solon-tls-certs"},
							},
						},
					},
				},
			},
		},
	}

	require.NoError(t, handler.checkForAnnotations(deployment))

	require.Len(t, handler.PendingUpdates["solon"], 1)
	hostUpdate := handler.PendingUpdates["solon"][0]
	assert.True(t, hostUpdate.IsHostUpdate)
	assert.Equal(t, "solon-tls-certs", hostUpdate.SecretName)
	assert.Equal(t, "delphi", hostUpdate.Namespace)
	assert.Equal(t, "Deployment", hostUpdate.KubeType)
	assert.Equal(t, 365, hostUpdate.Validity)

	require.Len(t, handler.PendingUpdates["aristarchos"], 1)
	assert.Equal(t, "solon", handler.PendingUpdates["aristarchos"][0].ClientName)
	require.Len(t, handler.PendingUpdates["herodotos"], 1)
	assert.Equal(t, "solon", handler.PendingUpdates["herodotos"][0].ClientName)
}

func TestCheckForAnnotationsUsesSecretFallback(t *testing.T) {
	handler := &PeriklesHandler{PendingUpdates: make(map[string][]MappingUpdate)}
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "eupalinos", Namespace: "agora"},
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						AnnotationHost:     "eupalinos",
						AnnotationValidity: "60",
					},
				},
			},
		},
	}

	require.NoError(t, handler.checkForAnnotations(deployment))
	require.Len(t, handler.PendingUpdates["eupalinos"], 1)
	assert.Equal(t, "eupalinos-tls-certs", handler.PendingUpdates["eupalinos"][0].SecretName)
}

func TestCheckForAnnotationsRejectsInvalidValidity(t *testing.T) {
	handler := &PeriklesHandler{PendingUpdates: make(map[string][]MappingUpdate)}
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "solon", Namespace: "delphi"},
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						AnnotationHost:     "solon",
						AnnotationValidity: "one year",
					},
				},
			},
		},
	}

	assert.Error(t, handler.checkForAnnotations(deployment))
	assert.Empty(t, handler.PendingUpdates)
}
