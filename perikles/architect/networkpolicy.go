package architect

import (
	"context"
	"fmt"
	ciliumv2 "github.com/cilium/cilium/pkg/k8s/apis/cilium.io/v2"
	"github.com/odysseia-greek/agora/plato/config"
	"github.com/odysseia-greek/agora/plato/logging"
	v1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	v2 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func (p *PeriklesHandler) checkForElasticAnnotations(deployment *v1.Deployment, job *batchv1.Job) error {
	var access, role string

	annotations := map[string]string{}
	var accessToServices string
	var kubeObject string
	var namespace string
	var containers []v2.Container

	if deployment != nil {
		annotations = deployment.Spec.Template.Annotations
		kubeObject = deployment.Name
		namespace = deployment.Namespace
		containers = deployment.Spec.Template.Spec.Containers
	}

	if job != nil {
		annotations = job.Spec.Template.Annotations
		kubeObject = job.Name
		namespace = job.Namespace
		containers = job.Spec.Template.Spec.Containers
	}

	for key, value := range annotations {
		if key == config.DefaultAccessAnnotation {
			access = value
		}

		if key == config.DefaultRoleAnnotation {
			role = value
		}

		if key == AnnotationAccesses {
			accessToServices = value
		}
	}

	if accessToServices != "" {
		p.generateServiceToServiceNetworkPolicy(kubeObject, namespace, accessToServices, containers)
	}

	if access == "" || role == "" {
		logging.Warn(fmt.Sprintf("No access annotations found in kube object %s/%s", namespace, kubeObject))
		return nil
	}

	policy := p.generateCiliumNetworkPolicyElastic(deployment, job, access, role)
	err := p.applyNetworkPolicy(policy)
	if err != nil {
		return err
	}

	return nil
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

	// Get the network policy first and delete if it exists
	_, err = p.Config.Kube.Dynamic().Resource(gvr).Namespace(policy.Namespace).Get(
		context.Background(),
		policy.Name,
		metav1.GetOptions{},
	)

	if err == nil {
		logging.Debug(fmt.Sprintf("CiliumNetworkPolicy %s found", policy.Name))

		err = p.Config.Kube.Dynamic().Resource(gvr).Namespace(policy.Namespace).Delete(
			context.Background(),
			policy.Name,
			metav1.DeleteOptions{},
		)

		logging.Debug(fmt.Sprintf("CiliumNetworkPolicy %s deleted", policy.Name))
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
				if value, ok := matchLabels["app:kubernetes.io/name"]; ok {
					matchLabels["app.kubernetes.io/name"] = value
					delete(matchLabels, "app:kubernetes.io/name")
				}

				if value, ok := matchLabels["any:app"]; ok {
					matchLabels["app"] = value
					delete(matchLabels, "any:app")
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
