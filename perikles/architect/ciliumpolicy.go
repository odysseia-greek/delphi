package architect

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	ciliumv2 "github.com/cilium/cilium/pkg/k8s/apis/cilium.io/v2"
	slimmetav1 "github.com/cilium/cilium/pkg/k8s/slim/k8s/apis/meta/v1"
	"github.com/cilium/cilium/pkg/policy/api"
	"github.com/odysseia-greek/agora/plato/logging"
	v1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	v2 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (p *PeriklesHandler) generateVaultNetworkPolicy(name, srcAppName, namespace string) error {
	newAnnotation := make(map[string]string)
	newAnnotation[AnnotationUpdate] = time.Now().UTC().Format(timeFormat)
	newAnnotation[IgnoreInGitOps] = "true"

	logging.Debug(fmt.Sprintf("adding vault policy for %s with app: %s in ns: %s", name, srcAppName, namespace))
	vaultPolicy := ciliumv2.CiliumNetworkPolicy{
		TypeMeta: metav1.TypeMeta{
			Kind:       "CiliumNetworkPolicy",
			APIVersion: "cilium.io/v2",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        fmt.Sprintf("allow-%s-access-vault", name),
			Namespace:   p.VaultNs,
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
									MatchLabels: map[string]slimmetav1.MatchLabelsValue{"app": srcAppName, "io.kubernetes.pod.namespace": namespace},
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
	err := p.applyNetworkPolicy(&vaultPolicy, p.VaultNs)
	if err != nil {
		return err
	}

	return nil
}
func (p *PeriklesHandler) generateServiceToServiceNetworkPolicy(name, namespace, hostsAnnotation, kubeType string, containers []v2.Container) {
	newAnnotation := make(map[string]string)
	newAnnotation[AnnotationUpdate] = time.Now().UTC().Format(timeFormat)
	newAnnotation[IgnoreInGitOps] = "true"

	var hosts []string
	if strings.Contains(hostsAnnotation, ";") {
		hosts = strings.Split(hostsAnnotation, ";")
	} else {
		hosts = []string{hostsAnnotation}
	}

	srcAppName, err := p.resolveAppSelectorName(name, namespace, kubeType)
	if err != nil {
		logging.Error(err.Error())
		return
	}

	if name != srcAppName {
		logging.Debug(fmt.Sprintf("label name %s does not match name %s so using label name %s instead", name, srcAppName, name))
	}

	for _, host := range hosts {
		ports, hostNamespace, err := p.findServicePortsForDeployment(host)
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
				Namespace:   hostNamespace,
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
										MatchLabels: map[string]slimmetav1.MatchLabelsValue{"app": srcAppName,
											"io.kubernetes.pod.namespace": namespace},
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

		err = p.applyNetworkPolicy(&policy, hostNamespace)
		if err != nil {
			logging.Error(err.Error())
		}
	}

	for _, container := range containers {
		if container.Name == "aristides" {
			logging.Debug(fmt.Sprintf("container found in deploy %s that requires vault access so adding np", name))
			err := p.generateVaultNetworkPolicy(name, srcAppName, namespace)
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

	var name, namespace, kubeType string

	if deploy == nil && job == nil {
		return nil
	}

	if deploy != nil {
		name = deploy.Name
		namespace = deploy.Namespace
		kubeType = "deployment"
		newAnnotation[AnnotationSourceKind] = "Deployment"
		newAnnotation[AnnotationSourceName] = deploy.Name
		newAnnotation[AnnotationSourceNamespace] = deploy.Namespace
		newAnnotation[AnnotationSourceUID] = string(deploy.UID)
	}

	if job != nil {
		name = job.Name
		namespace = job.Namespace
		kubeType = "job"
		newAnnotation[AnnotationSourceKind] = "Job"
		newAnnotation[AnnotationSourceName] = job.Name
		newAnnotation[AnnotationSourceNamespace] = job.Namespace
		newAnnotation[AnnotationSourceUID] = string(job.UID)
	}

	srcAppName, err := p.resolveAppSelectorName(name, namespace, kubeType)
	if err != nil {
		logging.Error(err.Error())
		return nil
	}

	if name != srcAppName {
		logging.Debug(fmt.Sprintf("label name %s does not match name %s so using label name %s instead", name, srcAppName, name))
	}

	// Define the CiliumNetworkPolicy
	policy := ciliumv2.CiliumNetworkPolicy{
		TypeMeta: metav1.TypeMeta{
			Kind:       "CiliumNetworkPolicy",
			APIVersion: "cilium.io/v2",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        fmt.Sprintf("restrict-elasticsearch-access-%s", name),
			Namespace:   p.ElasticNs,
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
									MatchLabels: map[string]slimmetav1.MatchLabelsValue{"app": srcAppName,
										"io.kubernetes.pod.namespace": namespace},
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

func (p *PeriklesHandler) findServicePortsForDeployment(deployName string) ([]v2.ServicePort, string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Search in watched namespaces first
	namespacesToSearch := append([]string{p.Namespace}, p.WatchedNamespaces...)

	for _, ns := range namespacesToSearch {
		deployment, err := p.Kube.AppsV1().Deployments(ns).Get(ctx, deployName, metav1.GetOptions{})
		if err == nil {
			ports, err := p.getServicePortsFromDeployment(deployment, ctx)
			return ports, deployment.Namespace, err
		}
	}

	// If not found, search in all namespaces
	namespaceList, err := p.Kube.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, "", fmt.Errorf("failed to list namespaces: %w", err)
	}

	// Skip namespaces we've already searched
	for _, ns := range namespaceList.Items {
		alreadySearched := false
		for _, searchedNs := range namespacesToSearch {
			if ns.Name == searchedNs {
				alreadySearched = true
				break
			}
		}

		if alreadySearched {
			continue
		}

		deployment, err := p.Kube.AppsV1().Deployments(ns.Name).Get(ctx, deployName, metav1.GetOptions{})
		if err == nil {
			ports, err := p.getServicePortsFromDeployment(deployment, ctx)
			return ports, deployment.Namespace, err
		}
	}

	return nil, "", fmt.Errorf("no deployment found with name %s in any namespace", deployName)
}

// Helper function to extract service ports from a deployment
func (p *PeriklesHandler) getServicePortsFromDeployment(deployment *v1.Deployment, ctx context.Context) ([]v2.ServicePort, error) {
	// Extract labels from the Deployment's pod template
	labels := deployment.Spec.Template.Labels
	if labels == nil {
		return nil, fmt.Errorf("deployment %s has no labels", deployment.Name)
	}

	// List Services in the deployment's namespace
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

func (p *PeriklesHandler) resolveAppSelectorName(objName, namespace, kubeType string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	var labels map[string]string

	switch kubeType {
	case "deployment":
		deploy, err := p.Kube.AppsV1().Deployments(namespace).Get(ctx, objName, metav1.GetOptions{})
		if err != nil {
			return "", err
		}
		labels = deploy.Labels

	case "job":
		job, err := p.Kube.BatchV1().Jobs(namespace).Get(ctx, objName, metav1.GetOptions{})
		if err != nil {
			return "", err
		}
		if app := appLabel(job.Spec.Template.Labels); app != "" {
			return app, nil
		}
		if app := appLabel(job.Labels); app != "" {
			return app, nil
		}

		for _, owner := range job.OwnerReferences {
			if owner.Kind != "CronJob" {
				continue
			}

			cronJob, err := p.Kube.BatchV1().CronJobs(namespace).Get(ctx, owner.Name, metav1.GetOptions{})
			if err != nil {
				return "", fmt.Errorf("resolve CronJob %s/%s for Job %s: %w", namespace, owner.Name, objName, err)
			}

			labelSets := []map[string]string{
				cronJob.Spec.JobTemplate.Spec.Template.Labels,
				cronJob.Spec.JobTemplate.Labels,
				cronJob.Labels,
			}
			for _, cronJobLabels := range labelSets {
				if app := appLabel(cronJobLabels); app != "" {
					return app, nil
				}
			}

			return cronJob.Name, nil
		}

		return objName, nil

	default:
		return "", fmt.Errorf("unsupported kubeType %q", kubeType)
	}

	if app, ok := labels["app"]; ok && app != "" {
		return app, nil
	}

	return objName, nil
}

func appLabel(labels map[string]string) string {
	if app, ok := labels["app"]; ok {
		return strings.TrimSpace(app)
	}
	return ""
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
