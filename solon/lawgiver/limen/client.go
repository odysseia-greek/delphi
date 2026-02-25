package limen

import (
	"fmt"

	"github.com/odysseia-greek/agora/aristoteles"
	"github.com/odysseia-greek/agora/diogenes"
	kubernetes "github.com/odysseia-greek/agora/thales"
	"github.com/odysseia-greek/delphi/solon/logoi"
	v1 "k8s.io/api/core/v1"
)

type Client struct {
	kubeClient *kubernetes.KubeClient
	namespaces logoi.Namespaces
	elastic    aristoteles.Client
	vault      diogenes.Client
}

func NewClient(
	kubeClient *kubernetes.KubeClient,
	namespaces logoi.Namespaces,
	elastic aristoteles.Client,
	vault diogenes.Client,
) *Client {
	return &Client{
		kubeClient: kubeClient,
		namespaces: namespaces,
		elastic:    elastic,
		vault:      vault,
	}
}

func (c *Client) deleteOrphans(pod *v1.Pod) error {
	if c.elastic == nil {
		return fmt.Errorf("limen client is not initialized with an elastic client")
	}
	if c.vault == nil {
		return fmt.Errorf("limen client is not initialized with a vault client")
	}
	return DeleteOrphans(c.elastic, c.vault, pod)
}
