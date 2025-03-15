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
	"strconv"
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

func (p *PeriklesHandler) generateVaultNetworkPolicy(name, namespace string) error {
	newAnnotation := make(map[string]string)
	newAnnotation[AnnotationUpdate] = time.Now().UTC().Format(timeFormat)
	newAnnotation[IgnoreInGitOps] = "true"

	vaultPolicy := ciliumv2.CiliumNetworkPolicy{
		TypeMeta: metav1.TypeMeta{
			Kind:       "CiliumNetworkPolicy",
			APIVersion: "cilium.io/v2",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        fmt.Sprintf("allow-%s-access-vault", name),
			Namespace:   namespace,
			Annotations: newAnnotation,
		},
		Spec: &api.Rule{
			// EndpointSelector selects the pods in the deployment
			EndpointSelector: api.EndpointSelector{
				LabelSelector: &slimmetav1.LabelSelector{
					MatchLabels: map[string]string{"app.kubernetes.io/name": "vault"},
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
									Port:     "8200",
									Protocol: api.ProtoTCP,
								},
							},
						},
					},
				},
			},
		},
	}
	err := p.applyNetworkPolicy(&vaultPolicy)
	if err != nil {
		return err
	}

	return nil
}
func (p *PeriklesHandler) generateServiceToServiceNetworkPolicy(name, namespace, hostsAnnotation string, containers []v2.Container) {
	newAnnotation := make(map[string]string)
	newAnnotation[AnnotationUpdate] = time.Now().UTC().Format(timeFormat)
	newAnnotation[IgnoreInGitOps] = "true"

	var hosts []string
	if strings.Contains(hostsAnnotation, ";") {
		hosts = strings.Split(hostsAnnotation, ";")
	} else {
		hosts = []string{hostsAnnotation}
	}

	for _, host := range hosts {
		ports, err := p.findServicePortsForDeployment(host, namespace)
		if err != nil {
			logging.Error(err.Error())
			continue
		}

		var portsOnHost []api.PortProtocol

		for _, port := range ports {
			portsOnHost = append(portsOnHost, api.PortProtocol{
				Port:     strconv.Itoa(int(port.Port)),
				Protocol: api.ProtoTCP,
			})
		}

		// Define the CiliumNetworkPolicy
		policy := ciliumv2.CiliumNetworkPolicy{
			TypeMeta: metav1.TypeMeta{
				Kind:       "CiliumNetworkPolicy",
				APIVersion: "cilium.io/v2",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:        fmt.Sprintf("allow-%s-access-%s", name, host),
				Namespace:   namespace,
				Annotations: newAnnotation,
			},
			Spec: &api.Rule{
				// EndpointSelector selects the pods in the deployment
				EndpointSelector: api.EndpointSelector{
					LabelSelector: &slimmetav1.LabelSelector{
						MatchLabels: map[string]string{"app": host},
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
								Ports: portsOnHost,
							},
						},
					},
				},
			},
		}

		err = p.applyNetworkPolicy(&policy)
		if err != nil {
			logging.Error(err.Error())
		}
	}

	for _, container := range containers {
		if container.Name == "aristides" {
			logging.Debug(fmt.Sprintf("container found in deploy %s that requires vault access so adding np", name))
			err := p.generateVaultNetworkPolicy(name, namespace)
			if err != nil {
				logging.Error(err.Error())
			}
		}
	}
}

func (p *PeriklesHandler) generateCiliumNetworkPolicyElastic(deploy *v1.Deployment, job *batchv1.Job, elasticAccess, role string) *ciliumv2.CiliumNetworkPolicy {
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

	if p.L7Mode {
		var indices []string

		if strings.Contains(elasticAccess, ";") {
			indices = strings.Split(elasticAccess, ";")
		} else {
			indices = []string{elasticAccess}
		}

		var rules []api.PortRuleHTTP
		for _, index := range indices {
			indexBasedRules := p.getHTTPRulesForRoleWithRegex(role, index)
			for _, rule := range indexBasedRules {
				rules = append(rules, rule)

			}
		}

		additionalPolicies := p.determineSideCars(containers, initContainers)

		for _, rule := range additionalPolicies {
			rules = append(rules, rule)
		}

		policy.Spec.Ingress[0].ToPorts[0].Rules = &api.L7Rules{
			HTTP: rules,
		}
	}

	return &policy
}

func (p *PeriklesHandler) getHTTPRulesForRoleWithRegex(role, index string) []api.PortRuleHTTP {
	var rules []api.PortRuleHTTP

	for _, rule := range p.RuleSet {
		if rule.RoleName == role {
			for _, cnp := range rule.CnpRules {
				rules = append(rules, api.PortRuleHTTP{
					Method: cnp.Method,
					Path:   strings.Replace(cnp.Path, "%%index%%", index, -1),
				})
			}
		}
	}

	healthEndpoint := api.PortRuleHTTP{Method: "^GET$", Path: "^/$"}
	rules = append(rules, healthEndpoint)

	return rules
}

func (p *PeriklesHandler) determineSideCars(containers []v2.Container, init []v2.Container) []api.PortRuleHTTP {
	var rules []api.PortRuleHTTP

	for _, initContainer := range init {
		if strings.Contains(initContainer.Name, "periandros") {
		}
	}

	for _, container := range containers {
		if strings.Contains(container.Name, "aristophanes") {
			// Update existing document in the aliased index
			rules = append(rules, api.PortRuleHTTP{
				Method: "^POST$",
				Path:   fmt.Sprintf("^/%s/_update/.*$", config.TracingElasticIndex), // Direct alias update
			})
			rules = append(rules, api.PortRuleHTTP{
				Method: "^POST$",
				Path:   fmt.Sprintf("^/%s-.*/_update/.*$", config.TracingElasticIndex), // Dynamic indices behind the alias
			})

			// Create a new document in the aliased index
			rules = append(rules, api.PortRuleHTTP{
				Method: "^PUT$",
				Path:   fmt.Sprintf("^/%s$", config.TracingElasticIndex), // Alias itself
			})
			rules = append(rules, api.PortRuleHTTP{
				Method: "^PUT$",
				Path:   fmt.Sprintf("^/%s/.*$", config.TracingElasticIndex), // Documents in the alias
			})
			rules = append(rules, api.PortRuleHTTP{
				Method: "^PUT$",
				Path:   fmt.Sprintf("^/%s-.*/.*$", config.TracingElasticIndex), // Dynamic indices behind the alias
			})
		}

	}
	return rules
}

func (p *PeriklesHandler) findServicePortsForDeployment(deployName, namespace string) ([]v2.ServicePort, error) {
	ctx := context.Background()

	// Get the Deployment
	deployment, err := p.Kube.AppsV1().Deployments(namespace).Get(ctx, deployName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get deployment %s: %w", deployName, err)
	}

	// Extract labels from the Deployment's pod template
	labels := deployment.Spec.Template.Labels
	if labels == nil {
		return nil, fmt.Errorf("deployment %s has no labels", deployment.Name)
	}

	// List Services in the namespace
	services, err := p.Kube.CoreV1().Services(deployment.Namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list services in namespace %s: %w", deployment.Namespace, err)
	}

	// Find the Service matching the Deployment's labels
	for _, service := range services.Items {
		if p.matchLabels(labels, service.Spec.Selector) {
			// Return the ports if a matching Service is found
			return service.Spec.Ports, nil
		}
	}

	return nil, fmt.Errorf("no service found for deployment %s", deployment.Name)
}

// Helper function to check if a Service selector matches Deployment labels
func (p *PeriklesHandler) matchLabels(deploymentLabels, serviceSelector map[string]string) bool {
	for key, value := range serviceSelector {
		if deploymentLabels[key] != value {
			return false
		}
	}
	return true
}
