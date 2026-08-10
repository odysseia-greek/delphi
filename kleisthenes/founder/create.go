package founder

type KleisthenesHandler struct {
	Kube            *KubeClient
	namespace       string
	periklesService string
}

func (k *KleisthenesHandler) Create() error {
	// if the services are already running no need to create any of these
	if err := k.perikles(); err != nil {
		return err
	}

	if err := k.vault(); err != nil {
		return err
	}

	return nil
}
