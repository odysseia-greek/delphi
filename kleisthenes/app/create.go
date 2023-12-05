package app

import (
	"github.com/odysseia-greek/agora/thales"
)

type KleisthenesHandler struct {
	Kube            *thales.KubeClient
	namespace       string
	periklesService string
}

func (k *KleisthenesHandler) Create() error {
	if err := k.perikles(); err != nil {
		return err
	}

	if err := k.vault(); err != nil {
		return err
	}

	return nil
}
