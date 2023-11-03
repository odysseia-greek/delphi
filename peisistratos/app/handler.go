package app

import (
	"embed"
	"fmt"
	"github.com/odysseia-greek/diogenes"
	plato "github.com/odysseia-greek/plato/config"
	"github.com/odysseia-greek/thales"
	"log"
	"os"
	"path/filepath"
	"strings"
)

type PeisistratosHandler struct {
	Pods      string
	Namespace string
	PodName   string
	Shares    int
	Threshold int
	Env       string
	Vault     diogenes.Client
	Kube      thales.KubeClient
}

const defaultAdminPolicyName = "solon"
const defaultUserPolicyName = "ptolemaios"

var (
	//go:embed hcl/policies
	embedPolicies embed.FS
)

func (p *PeisistratosHandler) InitVault() error {
	log.Print("init for vault in container started")

	status, err := p.Vault.Status()
	if err != nil {
		return err
	}

	if !status.Initialized {
		log.Print("vault is not initialized so starting there")

		var Nodes []string
		workload, err := p.Kube.Workload().GetStatefulSets(p.Namespace)
		if err != nil {
			return err
		}

		for _, item := range workload.Items {
			if item.Name == "vault" {
				var labelString string

				for key, value := range item.Labels {
					if key == "app.kubernetes.io/name" {
						labelString += fmt.Sprintf("%s=%s, ", key, value)
					}

				}

				// Remove the trailing comma and space
				labelString = labelString[:len(labelString)-2]

				pods, _ := p.Kube.Workload().GetPodsBySelector(p.Namespace, labelString)

				for _, pod := range pods.Items {
					Nodes = append(Nodes, pod.Name)
				}
			}
		}

		init, err := p.Vault.Initialize(p.Shares, p.Threshold)
		if err != nil {
			return err
		}

		log.Printf("vault is initialized root token: %s", init.RootToken)

		if len(Nodes) > 1 {

			rootPath := plato.StringFromEnv(plato.EnvRootTlSDir, plato.DefaultTLSFileLocation)
			secretPath := filepath.Join(rootPath, plato.VAULT)
			if p.Env == "LOCAL" {
				secretPath = "/tmp"
			}

			ca := fmt.Sprintf("%s/vault.ca", secretPath)
			cert := fmt.Sprintf("%s/vault.crt", secretPath)
			key := fmt.Sprintf("%s/vault.key", secretPath)

			_, err := p.Vault.Unseal(init.Keys)
			if err != nil {
				return err
			}

			readOutCa, _ := os.ReadFile(ca)
			readOutCert, _ := os.ReadFile(cert)
			readOutKey, _ := os.ReadFile(key)

			err = p.Vault.LoginWithRootToken(init.RootToken)
			if err != nil {
				return err
			}

			var PrimaryNode string
			leader, _ := p.Vault.Leader()
			for _, node := range Nodes {
				if strings.Contains(leader.LeaderClusterAddress, node) {
					PrimaryNode = node
				}
			}

			primaryAddress := fmt.Sprintf("https://%s.vault-internal:8200", PrimaryNode)

			for _, server := range Nodes {
				if p.Env == "LOCAL" {
					address := strings.Split(server, "-")[1]
					port := fmt.Sprintf("820%v", address)
					vaultPodDns := fmt.Sprintf("https://localhost:%s", port)

					tlsConfig := diogenes.CreateTLSConfig(true, ca, cert, key, secretPath)
					tempClient, err := diogenes.NewVaultClient(vaultPodDns, init.RootToken, tlsConfig)
					if err != nil {
						return err
					}

					if server != PrimaryNode {
						err = tempClient.LoginWithRootToken(init.RootToken)
						if err != nil {
							return err
						}

						raft, err := tempClient.RaftJoin(primaryAddress, readOutCert, readOutKey, readOutCa)
						if err != nil {
							return err
						}

						fmt.Print(raft.Joined)
					}

					_, err = tempClient.Unseal(init.Keys)
					if err != nil {
						return err
					}

				} else {
					vaultPodDns := fmt.Sprintf("https://%s.%s.svc.cluster.local:%v", server, p.Namespace, 8200)

					tlsConfig := diogenes.CreateTLSConfig(false, ca, cert, key, secretPath)
					tempClient, err := diogenes.NewVaultClient(vaultPodDns, init.RootToken, tlsConfig)
					if err != nil {
						return err
					}

					if server != PrimaryNode {
						err = tempClient.LoginWithRootToken(init.RootToken)
						if err != nil {
							return err
						}

						raft, err := tempClient.RaftJoin(primaryAddress, readOutCert, readOutKey, readOutCa)
						if err != nil {
							return err
						}

						log.Printf("joined raft: %v", raft.Joined)
					}

					unseal, err := tempClient.Unseal(init.Keys)
					if err != nil {
						return err
					}

					log.Printf("unsealed vault : %v", unseal)
				}
			}

		} else {
			unseal, err := p.Vault.Unseal(init.Keys)
			if err != nil {
				return err
			}

			log.Printf("unsealed vault : %v", unseal)
		}

		err = p.Vault.LoginWithRootToken(init.RootToken)

		err = p.Vault.EnableKVSecretsEngine("", "configs")
		if err != nil {
			return err
		}

		files, err := embedPolicies.ReadDir("hcl/policies")
		if err != nil {
			log.Fatal(err)
		}

		for _, file := range files {
			if file.IsDir() {
				continue // Skip directories
			}

			// Read the content of the HCL file
			content, err := embedPolicies.ReadFile(fmt.Sprintf("hcl/policies/%s", file.Name()))
			if err != nil {
				log.Printf("Error reading file %s: %v\n", file.Name(), err)
				continue
			}

			if strings.Contains(file.Name(), defaultAdminPolicyName) {
				err = p.Vault.WritePolicy(defaultAdminPolicyName, content)
				if err != nil {
					return err
				}
			} else {
				err = p.Vault.WritePolicy(defaultUserPolicyName, content)
				if err != nil {
					return err
				}
			}
		}

		kubeHostAddress := "https://kubernetes.default.svc"
		err = p.Vault.KubernetesAuthMethod(defaultAdminPolicyName, fmt.Sprintf("%s-access-sa", defaultAdminPolicyName), p.Namespace, kubeHostAddress)
		if err != nil {
			return err

		}
	}

	return nil
}
