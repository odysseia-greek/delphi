package architect

import (
	"fmt"
	"github.com/cilium/cilium/pkg/policy/api"
	"github.com/odysseia-greek/agora/plato/config"
	kubernetes "github.com/odysseia-greek/agora/thales"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	v2 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func TestCheckForElasticAnnotations(t *testing.T) {
	handler := PeriklesHandler{
		Config: &Config{
			Kube:   kubernetes.NewFakeKubeClient(),
			L7Mode: false,
		},
	}

	t.Run("ValidDeploymentAnnotations", func(t *testing.T) {
		deployment := &v1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-deployment",
				Namespace: "test-namespace",
			},
			Spec: v1.DeploymentSpec{
				Template: v2.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							config.DefaultAccessAnnotation: "access-value",
							config.DefaultRoleAnnotation:   "role-value",
						},
					},
				},
			},
		}

		err := handler.checkForElasticAnnotations(deployment, nil)
		assert.Nil(t, err)
	})

	t.Run("ValidJobAnnotations", func(t *testing.T) {
		job := &batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-job",
				Namespace: "test-namespace",
			},
			Spec: batchv1.JobSpec{
				Template: v2.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							config.DefaultAccessAnnotation: "access-value",
							config.DefaultRoleAnnotation:   "role-value",
						},
					},
				},
			},
		}

		err := handler.checkForElasticAnnotations(nil, job)
		assert.Nil(t, err)
	})

	t.Run("MissingAnnotations", func(t *testing.T) {
		deployment := &v1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-deployment",
				Namespace: "test-namespace",
			},
			Spec: v1.DeploymentSpec{
				Template: v2.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{}, // Empty annotations
					},
				},
			},
		}

		err := handler.checkForElasticAnnotations(deployment, nil)
		assert.NotNil(t, err)
	})
}

func TestGenerateCiliumNetworkPolicyInL3L4Mode(t *testing.T) {
	handler := PeriklesHandler{
		Config: &Config{
			L7Mode: false, // Ensure L7Mode is off
		},
	}

	t.Run("GeneratePolicyForDeploymentWithoutL7Rules", func(t *testing.T) {
		deployment := &v1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-deployment",
				Namespace: "test-namespace",
			},
		}

		policy := handler.generateCiliumNetworkPolicy(deployment, nil, "access-value", "role-value")
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

		policy := handler.generateCiliumNetworkPolicy(nil, job, "access-value", "role-value")
		assert.NotNil(t, policy)
		assert.Equal(t, "restrict-elasticsearch-access-test-job", policy.Name)
		assert.Equal(t, "test-namespace", policy.Namespace)

		// Check that no L7 rules are present
		assert.Nil(t, policy.Spec.Ingress[0].ToPorts[0].Rules)
	})

	t.Run("NilInputsWithoutL7Rules", func(t *testing.T) {
		policy := handler.generateCiliumNetworkPolicy(nil, nil, "access-value", "role-value")
		assert.Nil(t, policy)
	})
}

func TestGenerateCiliumNetworkPolicyInL7Mode(t *testing.T) {
	handler := PeriklesHandler{
		Config: &Config{
			L7Mode: true, // Enable L7 mode
		},
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

		policy := handler.generateCiliumNetworkPolicy(deployment, nil, "access-value", "role-value")
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

		policy := handler.generateCiliumNetworkPolicy(nil, job, "access-value", "role-value")
		assert.NotNil(t, policy)

		// Check that L7 rules are present
		assert.NotNil(t, policy.Spec.Ingress[0].ToPorts[0].Rules)
		assert.NotNil(t, policy.Spec.Ingress[0].ToPorts[0].Rules.HTTP)

		// Verify that rules for both containers and init containers are added
		rules := policy.Spec.Ingress[0].ToPorts[0].Rules.HTTP
		assert.True(t, len(rules) > 0)
	})

	t.Run("NilInputsWithL7Rules", func(t *testing.T) {
		policy := handler.generateCiliumNetworkPolicy(nil, nil, "access-value", "role-value")
		assert.Nil(t, policy)
	})
}

func TestL7RulesGeneration(t *testing.T) {
	handler := PeriklesHandler{
		Config: &Config{
			L7Mode: true, // Enable L7 mode for these tests
		},
	}

	t.Run("GetHTTPRulesForRoles", func(t *testing.T) {
		roles := map[string][]api.PortRuleHTTP{
			CreatorElasticRole: {
				{Method: "^GET$", Path: "^/$"}, // Default health check endpoint
			},
			SeederElasticRole: {
				{Method: "^PUT$", Path: "^/index/.*$"},
				{Method: "^POST$", Path: "^/index/_create$"},
				{Method: "^GET$", Path: "^/$"}, // Default health check endpoint
			},
			HybridElasticRole: {
				{Method: "^POST$", Path: "^/index/.*$"},
				{Method: "^PUT$", Path: "^/index/_create$"},
				{Method: "^GET$", Path: "^/$"}, // Default health check endpoint
			},
			ApiElasticRole: {
				{Method: "^POST$", Path: "^/index/.*$"},
				{Method: "^GET$", Path: "^/$"}, // Default health check endpoint
			},
			AliasElasticRole: {
				{Method: "^GET$", Path: "^/index/_search/??.*$"},
				{Method: "^POST$", Path: "^/index/.*$"},
				{Method: "^GET$", Path: "^/$"}, // Default health check endpoint
			},
		}

		for role, expectedRules := range roles {
			rules := handler.getHTTPRulesForRoleWithRegex(role, "index")
			assert.Equal(t, len(expectedRules), len(rules))
			for i, rule := range rules {
				assert.Equal(t, expectedRules[i].Method, rule.Method)
				assert.Equal(t, expectedRules[i].Path, rule.Path)
			}
		}
	})

	t.Run("DetermineSideCarRules", func(t *testing.T) {
		containers := []v2.Container{
			{Name: "aristophanes-tracing"},
			{Name: "sophokles-metrics"},
		}
		initContainers := []v2.Container{
			{Name: "init-pe-container"},
		}

		rules := handler.determineSideCars(containers, initContainers)

		expectedRules := []api.PortRuleHTTP{
			{Method: "POST", Path: fmt.Sprintf("^/%s/*", config.TracingElasticIndex)},
			{Method: "PUT", Path: fmt.Sprintf("^/%s/_create$", config.TracingElasticIndex)},
			{Method: "POST", Path: fmt.Sprintf("^/%s/*", config.MetricsElasticIndex)},
			{Method: "PUT", Path: fmt.Sprintf("^/%s/_create$", config.MetricsElasticIndex)},
		}

		assert.Equal(t, len(expectedRules), len(rules))
		for i, rule := range rules {
			assert.Equal(t, expectedRules[i].Method, rule.Method)
			assert.Equal(t, expectedRules[i].Path, rule.Path)
		}
	})
}
