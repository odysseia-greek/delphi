package architect

import (
	"os"
	"path/filepath"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type KubeClient struct {
	kubernetes.Interface
	config  *rest.Config
	dynamic dynamic.Interface
}

func newKubeClient() (*KubeClient, error) {
	config, err := kubeConfig()
	if err != nil {
		return nil, err
	}
	typed, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return &KubeClient{Interface: typed, config: config, dynamic: dynamicClient}, nil
}

func newFakeKubeClient() *KubeClient {
	return &KubeClient{Interface: fake.NewSimpleClientset(), config: &rest.Config{}, dynamic: dynamicfake.NewSimpleDynamicClient(runtime.NewScheme())}
}

func (c *KubeClient) RestConfig() *rest.Config   { return c.config }
func (c *KubeClient) Dynamic() dynamic.Interface { return c.dynamic }

func kubeConfig() (*rest.Config, error) {
	if config, err := rest.InClusterConfig(); err == nil {
		return config, nil
	}
	path := os.Getenv("KUBECONFIG")
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		path = filepath.Join(home, ".kube", "config")
	}
	return clientcmd.BuildConfigFromFlags("", path)
}
