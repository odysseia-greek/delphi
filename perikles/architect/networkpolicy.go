package architect

import (
	"context"
	"fmt"
	ciliumv2 "github.com/cilium/cilium/pkg/k8s/apis/cilium.io/v2"
	slimmetav1 "github.com/cilium/cilium/pkg/k8s/slim/k8s/apis/meta/v1"
	"github.com/cilium/cilium/pkg/policy/api"
	"github.com/odysseia-greek/agora/plato/config"
	"github.com/odysseia-greek/agora/plato/logging"
	v1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	v2 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"strings"
	"time"
)

const (
	CreatorElasticRole = "creator"
	SeederElasticRole  = "seeder"
	HybridElasticRole  = "hybrid"
	ApiElasticRole     = "api"
	AliasElasticRole   = "alias"
)

func (p *PeriklesHandler) checkForElasticAnnotations(deployment *v1.Deployment, job *batchv1.Job) error {
	var access, role string

	annotations := map[string]string{}
	if deployment != nil {
		annotations = deployment.Spec.Template.Annotations
	}

	if job != nil {
		annotations = job.Spec.Template.Annotations
	}
	for key, value := range annotations {
		if key == config.DefaultAccessAnnotation {
			access = value
		}

		if key == config.DefaultRoleAnnotation {
			role = value
		}
	}

	if access == "" || role == "" {
		return fmt.Errorf("no role or access annotations found in deployment")
	}
	policy := p.generateCiliumNetworkPolicy(deployment, job, access, role)
	err := p.applyNetworkPolicy(policy)
	if err != nil {
		return err
	}

	return nil
}

func (p *PeriklesHandler) generateCiliumNetworkPolicy(deploy *v1.Deployment, job *batchv1.Job, elasticAccess, role string) *ciliumv2.CiliumNetworkPolicy {
	// Define the annotations for tracking policy creation
	newAnnotation := make(map[string]string)
	newAnnotation[AnnotationUpdate] = time.Now().UTC().Format(timeFormat)
	newAnnotation[IgnoreInGitOps] = "true"

	var name, namespace string
	var initContainers, containers []v2.Container

	if deploy == nil && job == nil {
		return nil
	}

	if deploy != nil {
		name = deploy.Name
		namespace = deploy.Namespace
		initContainers = deploy.Spec.Template.Spec.InitContainers
		containers = deploy.Spec.Template.Spec.Containers
	}

	if job != nil {
		name = job.Name
		namespace = job.Namespace
		initContainers = job.Spec.Template.Spec.InitContainers
		containers = job.Spec.Template.Spec.Containers
	}

	// Define the CiliumNetworkPolicy
	policy := ciliumv2.CiliumNetworkPolicy{
		TypeMeta: metav1.TypeMeta{
			Kind:       "CiliumNetworkPolicy",
			APIVersion: "cilium.io/v2",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        fmt.Sprintf("restrict-elasticsearch-access-%s", name),
			Namespace:   namespace,
			Annotations: newAnnotation,
		},
		Spec: &api.Rule{
			// EndpointSelector selects the pods in the deployment
			EndpointSelector: api.EndpointSelector{
				LabelSelector: &slimmetav1.LabelSelector{
					MatchLabels: map[string]string{"elasticsearch.k8s.elastic.co/cluster-name": "aristoteles"},
				},
			},
			// Define Ingress rules with IngressCommonRule
			Ingress: []api.IngressRule{
				{
					IngressCommonRule: api.IngressCommonRule{
						FromEndpoints: []api.EndpointSelector{
							{
								LabelSelector: &slimmetav1.LabelSelector{
									MatchLabels: map[string]slimmetav1.MatchLabelsValue{"app": name},
								},
							},
						},
					},
					ToPorts: []api.PortRule{
						{
							Ports: []api.PortProtocol{
								{
									Port:     "9200",
									Protocol: api.ProtoTCP,
								},
							},
						},
					},
				},
			},
		},
	}

	if p.Config.L7Mode {
		rules := p.getHTTPRulesForRoleWithRegex(role, elasticAccess)

		additionalPolicies := p.determineSideCars(initContainers, containers)

		for _, rule := range additionalPolicies {
			rules = append(rules, rule)
		}

		policy.Spec.Ingress[0].ToPorts[0].Rules = &api.L7Rules{
			HTTP: rules,
		}
	}

	return &policy
}

// applyNetworkPolicy applies a CiliumNetworkPolicy using the dynamic Kubernetes client.
func (p *PeriklesHandler) applyNetworkPolicy(policy *ciliumv2.CiliumNetworkPolicy) error {
	// Define the GVR (GroupVersionResource) for the CiliumNetworkPolicy
	gvr := schema.GroupVersionResource{
		Group:    "cilium.io",
		Version:  "v2",
		Resource: "ciliumnetworkpolicies",
	}

	// Convert the CiliumNetworkPolicy to an unstructured object
	unstructuredObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&policy)
	if err != nil {
		return fmt.Errorf("failed to convert CiliumNetworkPolicy to unstructured: %w", err)
	}

	// Correct the problematic fields in the unstructured object
	if spec, found, _ := unstructured.NestedMap(unstructuredObj, "spec"); found {
		// Fix `endpointSelector.matchLabels` (for elasticsearch key)
		if endpointSelector, found, _ := unstructured.NestedMap(spec, "endpointSelector"); found {
			if matchLabels, found, _ := unstructured.NestedMap(endpointSelector, "matchLabels"); found {
				if value, ok := matchLabels["elasticsearch:k8s.elastic.co/cluster-name"]; ok {
					matchLabels["elasticsearch.k8s.elastic.co/cluster-name"] = value
					delete(matchLabels, "elasticsearch:k8s.elastic.co/cluster-name")
				}
				_ = unstructured.SetNestedMap(endpointSelector, matchLabels, "matchLabels")
			}
			_ = unstructured.SetNestedMap(spec, endpointSelector, "endpointSelector")
		}

		// Fix `fromEndpoints.matchLabels` (for any:app key)
		if ingress, found, _ := unstructured.NestedSlice(spec, "ingress"); found {
			for i, rule := range ingress {
				if ruleMap, ok := rule.(map[string]interface{}); ok {
					if fromEndpoints, found, _ := unstructured.NestedSlice(ruleMap, "fromEndpoints"); found {
						for j, endpoint := range fromEndpoints {
							if endpointMap, ok := endpoint.(map[string]interface{}); ok {
								if matchLabels, found, _ := unstructured.NestedMap(endpointMap, "matchLabels"); found {
									if value, ok := matchLabels["any:app"]; ok {
										matchLabels["app"] = value
										delete(matchLabels, "any:app")
									}
									_ = unstructured.SetNestedMap(endpointMap, matchLabels, "matchLabels")
								}
								fromEndpoints[j] = endpointMap
							}
						}
						_ = unstructured.SetNestedSlice(ruleMap, fromEndpoints, "fromEndpoints")
					}
					ingress[i] = ruleMap
				}
			}
			_ = unstructured.SetNestedSlice(spec, ingress, "ingress")
		}

		// Update the unstructured object with the corrected spec
		_ = unstructured.SetNestedMap(unstructuredObj, spec, "spec")
	}

	// Wrap the corrected unstructured object
	unstructuredPolicy := &unstructured.Unstructured{Object: unstructuredObj}

	// Apply the policy in the specified namespace
	_, err = p.Config.Kube.Dynamic().Resource(gvr).Namespace(policy.Namespace).Create(
		context.Background(),
		unstructuredPolicy,
		metav1.CreateOptions{},
	)
	if err != nil {
		return fmt.Errorf("failed to apply CiliumNetworkPolicy in namespace %s: %w", policy.Namespace, err)
	}

	logging.Debug(fmt.Sprintf("Successfully applied CiliumNetworkPolicy %s in namespace %s", policy.Name, policy.Namespace))
	return nil
}

func (p *PeriklesHandler) getHTTPRulesForRoleWithRegex(role, index string) []api.PortRuleHTTP {
	var rules []api.PortRuleHTTP

	switch role {
	case SeederElasticRole:
		rules = append(rules, api.PortRuleHTTP{Method: "^PUT$", Path: fmt.Sprintf("^/%s/.*$", index)})
		rules = append(rules, api.PortRuleHTTP{Method: "^POST$", Path: fmt.Sprintf("^/%s/_create$", index)})

	case HybridElasticRole:
		rules = append(rules, api.PortRuleHTTP{Method: "^POST$", Path: fmt.Sprintf("^/%s/.*$", index)})
		rules = append(rules, api.PortRuleHTTP{Method: "^PUT$", Path: fmt.Sprintf("^/%s/_create$", index)})

	case ApiElasticRole:
		rules = append(rules, api.PortRuleHTTP{Method: "^POST$", Path: fmt.Sprintf("^/%s/.*$", index)})

	case AliasElasticRole:
		rules = append(rules, api.PortRuleHTTP{Method: "^GET$", Path: fmt.Sprintf("^/%s/_search/??.*$", index)})
		rules = append(rules, api.PortRuleHTTP{Method: "^POST$", Path: fmt.Sprintf("^/%s/.*$", index)})
	}

	healthEndpoint := api.PortRuleHTTP{Method: "^GET$", Path: "^/$"}
	rules = append(rules, healthEndpoint)

	return rules
}

func (p *PeriklesHandler) determineSideCars(containers []v2.Container, init []v2.Container) []api.PortRuleHTTP {
	var rules []api.PortRuleHTTP

	for _, initContainer := range init {
		if strings.Contains(initContainer.Name, "pe") {

		}
	}

	for _, container := range containers {
		if strings.Contains(container.Name, "aristophanes") {
			rules = append(rules, api.PortRuleHTTP{Method: "POST", Path: fmt.Sprintf("^/%s/*", config.TracingElasticIndex)})       // For indexing and creating
			rules = append(rules, api.PortRuleHTTP{Method: "PUT", Path: fmt.Sprintf("^/%s/_create$", config.TracingElasticIndex)}) // For direct document creation
		}

		if strings.Contains(container.Name, "sophokles") {
			rules = append(rules, api.PortRuleHTTP{Method: "POST", Path: fmt.Sprintf("^/%s/*", config.MetricsElasticIndex)})       // For indexing and creating
			rules = append(rules, api.PortRuleHTTP{Method: "PUT", Path: fmt.Sprintf("^/%s/_create$", config.MetricsElasticIndex)}) // For direct document creation
		}
	}
	return rules
}
