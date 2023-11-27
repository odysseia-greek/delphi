package app

import (
	"embed"
	"encoding/json"
	"fmt"
	"github.com/hashicorp/vault/api"
	"github.com/odysseia-greek/diogenes"
	plato "github.com/odysseia-greek/plato/config"
	"github.com/odysseia-greek/plato/logging"
	"github.com/odysseia-greek/thales"
	"log"
	"os"
	"path/filepath"
	"strings"
)

type PeisistratosHandler struct {
	Pods         string
	Namespace    string
	PodName      string
	Shares       int
	Threshold    int
	Env          string
	UnsealMethod string
	Vault        diogenes.Client
	Kube         thales.KubeClient
}

const defaultAdminPolicyName = "solon"
const defaultUserPolicyName = "ptolemaios"
const gcp = "gcp"

var (
	//go:embed hcl/policies
	embedPolicies embed.FS
)

func (p *PeisistratosHandler) InitVault() error {
	logging.Debug("init for vault in container started")

	status, err := p.Vault.Status()
	if err != nil {
		return err
	}

	jsonStatus, err := json.MarshalIndent(status, "", "\t")
	if err != nil {
		return err
	}
	logging.Debug(fmt.Sprintf("vault status: %s", jsonStatus))

	if status.Initialized {
		return nil
	}

	logging.Debug("vault is not initialized so first step is initializing it")

	nodes, err := p.getVaultPodNodes()
	if err != nil {
		return err
	}

	var init *api.InitResponse

	if p.UnsealMethod != "" {
		logging.Info("initializing vault with auto unseal")
		init, err = p.Vault.InitializeAutoUnseal(1)
		if err != nil {
			return err
		}
	} else {
		logging.Info("initializing vault without auto unseal")
		init, err = p.Vault.Initialize(p.Shares, p.Threshold)
		if err != nil {
			return err
		}

		logging.Debug(fmt.Sprintf("vault is initialized with the following shares: %s", init.Keys))
	}

	logging.Debug(fmt.Sprintf("vault is initialized root token: %s", init.RootToken))

	if len(nodes) > 1 {
		err := p.haFlow(nodes, init)
		if err != nil {
			return err
		}

	} else {
		err := p.unsealVault(init)
		if err != nil {
			return err
		}
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

	return nil
}

func (p *PeisistratosHandler) haFlow(nodes []string, init *api.InitResponse) error {
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
	for _, node := range nodes {
		if strings.Contains(leader.LeaderClusterAddress, node) {
			PrimaryNode = node
		}
	}

	primaryAddress := fmt.Sprintf("https://%s.vault-internal:8200", PrimaryNode)

	for _, server := range nodes {
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

				logging.Debug(fmt.Sprintf("raft joined: %v", raft.Joined))
			}

			err = p.unsealVault(init)
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

				logging.Debug(fmt.Sprintf("raft joined: %v", raft.Joined))
			}

			err = p.unsealVault(init)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (p *PeisistratosHandler) unsealVault(init *api.InitResponse) error {
	var unseal bool
	var err error
	switch p.UnsealMethod {
	case gcp:
		config := createUnsealConfig(gcp).(GCPConfig)
		unseal, err = p.Vault.AutoUnsealGCP(config.KeyRing, config.CryptoKey, config.Location, init.RecoveryKeys)
		if err != nil {
			return err
		}

	default:
		unseal, err = p.Vault.Unseal(init.Keys)
		if err != nil {
			return err
		}
	}

	logging.Debug(fmt.Sprintf("unsealed vault : %v", unseal))
	return nil
}

func (p *PeisistratosHandler) getVaultPodNodes() ([]string, error) {
	var nodes []string
	workload, err := p.Kube.Workload().GetStatefulSets(p.Namespace)
	if err != nil {
		return nil, err
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
				nodes = append(nodes, pod.Name)
			}
		}
	}

	return nodes, nil
}

type UnsealConfig interface{}

type GCPConfig struct {
	KeyRing   string
	CryptoKey string
	Location  string
}

type AzureConfig struct {
}

func createUnsealConfig(provider string) UnsealConfig {
	switch provider {
	case gcp:
		config := GCPConfig{
			KeyRing:   os.Getenv("KEY_RING"),
			CryptoKey: os.Getenv("CRYPTO_KEY"),
			Location:  os.Getenv("LOCATION"),
		}
		return config

	case "azure":
		config := AzureConfig{}
		return config
	}

	return nil
}
