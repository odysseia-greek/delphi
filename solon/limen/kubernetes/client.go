package kubernetes

import (
	"context"
	"os"
	"path/filepath"

	"github.com/odysseia-greek/delphi/solon/logoi"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type Cleaner interface {
	DeleteOrphan(ctx context.Context, username, podName string) error
}

type Client struct {
	kubernetes.Interface
	namespaces logoi.Namespaces
	cleaner    Cleaner
}

func NewClient(namespaces logoi.Namespaces, cleaner Cleaner) (*Client, error) {
	config, err := kubeConfig()
	if err != nil {
		return nil, err
	}
	typed, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return newClient(typed, namespaces, cleaner), nil
}

func NewFakeClient(namespaces logoi.Namespaces, cleaner Cleaner) *Client {
	return newClient(fake.NewSimpleClientset(), namespaces, cleaner)
}

func newClient(typed kubernetes.Interface, namespaces logoi.Namespaces, cleaner Cleaner) *Client {
	return &Client{Interface: typed, namespaces: namespaces, cleaner: cleaner}
}

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
