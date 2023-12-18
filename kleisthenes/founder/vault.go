package founder

import (
	"fmt"
	"github.com/odysseia-greek/agora/plato/certificates"
	"github.com/odysseia-greek/agora/plato/logging"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
)

func (k *KleisthenesHandler) vault() error {
	logging.Debug("Setting up TLS for Vault")

	validity := 3650 // valid for 10 years

	orgName := []string{
		k.namespace,
	}

	hosts := []string{
		fmt.Sprintf("%s", "vault"),
		fmt.Sprintf("%s.%s", "vault", k.namespace),
		fmt.Sprintf("%s.%s.svc", "vault", k.namespace),
		fmt.Sprintf("%s.%s.svc.cluster.local", "vault", k.namespace),
		fmt.Sprintf("%s-0.vault-internal", "vault"),
		fmt.Sprintf("%s-1.vault-internal", "vault"),
		fmt.Sprintf("%s-2.vault-internal", "vault"),
	}

	certClient, err := certificates.NewCertGeneratorClient(orgName, validity)
	if err != nil {
		return err
	}

	err = certClient.InitCa()
	if err != nil {
		return err
	}

	cert, key, err := certClient.GenerateKeyAndCertSet(hosts, validity)
	if err != nil {
		return err
	}

	ca := certClient.PemEncodedCa()

	secretName := "vault-server-tls"
	data := make(map[string][]byte)
	data["vault.crt"] = cert
	data["vault.key"] = key
	data["vault.ca"] = ca

	logging.Debug(fmt.Sprintf("vault.crt: %s", string(cert)))
	logging.Debug(fmt.Sprintf("vault.ca: %s", string(ca)))
	logging.Debug(fmt.Sprintf("vault.key: %s", string(key)))

	if err := k.createSecret(secretName, data, corev1.SecretTypeOpaque, false); err != nil {
		return errors.Wrap(err, "failed to create secret")
	}

	logging.Debug("Created secret")
	logging.Debug("Finished setting up TLS for Vault")
	return nil
}
