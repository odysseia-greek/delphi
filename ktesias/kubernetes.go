package ktesias

import (
	"os"
	"path/filepath"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type KubeClient struct {
	kubernetes.Interface
	config *rest.Config
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
	return &KubeClient{Interface: typed, config: config}, nil
}

func (c *KubeClient) RestConfig() *rest.Config { return c.config }

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
