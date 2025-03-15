package architect

import (
	kubernetes "github.com/odysseia-greek/agora/thales"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	v2 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func TestGenerateCiliumNetworkPolicyInL3L4Mode(t *testing.T) {
	handler := PeriklesHandler{
		L7Mode: false, // Ensure L7Mode is off
	}

	t.Run("GeneratePolicyForDeploymentWithoutL7Rules", func(t *testing.T) {
		deployment := &v1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-deployment",
				Namespace: "test-namespace",
			},
		}

		policy := handler.generateCiliumNetworkPolicyElastic(deployment, nil, "access-value", "role-value")
		assert.NotNil(t, policy)
		assert.Equal(t, "restrict-elasticsearch-access-test-deployment", policy.Name)
		assert.Equal(t, "test-namespace", policy.Namespace)

		// Check that no L7 rules are present
		assert.Nil(t, policy.Spec.Ingress[0].ToPorts[0].Rules)
	})

	t.Run("GeneratePolicyForJobWithoutL7Rules", func(t *testing.T) {
		job := &batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-job",
				Namespace: "test-namespace",
			},
		}

		policy := handler.generateCiliumNetworkPolicyElastic(nil, job, "access-value", "role-value")
		assert.NotNil(t, policy)
		assert.Equal(t, "restrict-elasticsearch-access-test-job", policy.Name)
		assert.Equal(t, "test-namespace", policy.Namespace)

		// Check that no L7 rules are present
		assert.Nil(t, policy.Spec.Ingress[0].ToPorts[0].Rules)
	})

	t.Run("NilInputsWithoutL7Rules", func(t *testing.T) {
		policy := handler.generateCiliumNetworkPolicyElastic(nil, nil, "access-value", "role-value")
		assert.Nil(t, policy)
	})
}

func TestGenerateCiliumNetworkPolicyInL7Mode(t *testing.T) {
	handler := PeriklesHandler{
		L7Mode: true, // Enable L7 mode
	}

	t.Run("GeneratePolicyWithL7RulesForDeployment", func(t *testing.T) {
		deployment := &v1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-deployment",
				Namespace: "test-namespace",
			},
			Spec: v1.DeploymentSpec{
				Template: v2.PodTemplateSpec{
					Spec: v2.PodSpec{
						InitContainers: []v2.Container{
							{Name: "init-container-1"},
						},
						Containers: []v2.Container{
							{Name: "container-1"},
						},
					},
				},
			},
		}

		policy := handler.generateCiliumNetworkPolicyElastic(deployment, nil, "access-value", "role-value")
		assert.NotNil(t, policy)

		// Check that L7 rules are present
		assert.NotNil(t, policy.Spec.Ingress[0].ToPorts[0].Rules)
		assert.NotNil(t, policy.Spec.Ingress[0].ToPorts[0].Rules.HTTP)

		// Verify that rules for both containers and init containers are added
		rules := policy.Spec.Ingress[0].ToPorts[0].Rules.HTTP
		assert.True(t, len(rules) > 0)
	})

	t.Run("GeneratePolicyWithL7RulesForJob", func(t *testing.T) {
		job := &batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-job",
				Namespace: "test-namespace",
			},
			Spec: batchv1.JobSpec{
				Template: v2.PodTemplateSpec{
					Spec: v2.PodSpec{
						InitContainers: []v2.Container{
							{Name: "init-container-2"},
						},
						Containers: []v2.Container{
							{Name: "container-2"},
						},
					},
				},
			},
		}

		policy := handler.generateCiliumNetworkPolicyElastic(nil, job, "access-value", "role-value")
		assert.NotNil(t, policy)

		// Check that L7 rules are present
		assert.NotNil(t, policy.Spec.Ingress[0].ToPorts[0].Rules)
		assert.NotNil(t, policy.Spec.Ingress[0].ToPorts[0].Rules.HTTP)

		// Verify that rules for both containers and init containers are added
		rules := policy.Spec.Ingress[0].ToPorts[0].Rules.HTTP
		assert.True(t, len(rules) > 0)
	})

	t.Run("NilInputsWithL7Rules", func(t *testing.T) {
		policy := handler.generateCiliumNetworkPolicyElastic(nil, nil, "access-value", "role-value")
		assert.Nil(t, policy)
	})
}

// best to create an integration test here at some point using KWOK
func TestGenerateVaultNetworkPolicy(t *testing.T) {
	handler := PeriklesHandler{
		Kube:   kubernetes.NewFakeKubeClient(),
		L7Mode: false,
	}

	t.Run("GenerateValidVaultPolicy", func(t *testing.T) {
		err := handler.generateVaultNetworkPolicy("test-app", "test-namespace")
		assert.Nil(t, err)
	})
}

// best to create an integration test here at some point using KWOK
func TestGenerateServiceToServiceNetworkPolicy(t *testing.T) {
	handler := PeriklesHandler{
		Kube:   kubernetes.NewFakeKubeClient(),
		L7Mode: false,
	}

	t.Run("GeneratePolicyWithSingleHost", func(t *testing.T) {
		containers := []v2.Container{
			{Name: "test-container"},
		}

		hostsAnnotation := "test-host"
		handler.generateServiceToServiceNetworkPolicy("test-app", "test-namespace", hostsAnnotation, containers)
	})

	t.Run("GeneratePolicyWithMultipleHosts", func(t *testing.T) {
		containers := []v2.Container{
			{Name: "test-container"},
		}

		hostsAnnotation := "test-host1;test-host2"
		handler.generateServiceToServiceNetworkPolicy("test-app", "test-namespace", hostsAnnotation, containers)
	})

	t.Run("GeneratePolicyWithVaultAccess", func(t *testing.T) {
		containers := []v2.Container{
			{Name: "aristides"},
		}

		hostsAnnotation := "test-host"
		handler.generateServiceToServiceNetworkPolicy("test-app", "test-namespace", hostsAnnotation, containers)
	})
}
