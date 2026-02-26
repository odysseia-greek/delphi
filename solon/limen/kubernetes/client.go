package kubernetes

import (
	kubernetes "github.com/odysseia-greek/agora/thales"
	"github.com/odysseia-greek/delphi/solon/logoi"
)

type Cleaner interface {
	DeleteOrphan(username, podName string) error
}

type Client struct {
	kubeClient *kubernetes.KubeClient
	namespaces logoi.Namespaces
	cleaner    Cleaner
}

func NewClient(
	kubeClient *kubernetes.KubeClient,
	namespaces logoi.Namespaces,
	cleaner Cleaner,
) *Client {
	return &Client{
		kubeClient: kubeClient,
		namespaces: namespaces,
		cleaner:    cleaner,
	}
}
