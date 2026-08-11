package architect

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hashicorp/vault/api"
	"github.com/odysseia-greek/agora/diogenes"
	plato "github.com/odysseia-greek/agora/plato/config"
	"github.com/odysseia-greek/agora/plato/logging"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type PeisistratosHandler struct {
	Pods         string
	Namespace    string
	PodName      string
	Shares       int
	Threshold    int
	Env          string
	Vault        diogenes.Client
	Kube         *KubeClient
	UnsealMethod string
}

const (
	defaultAdminPolicyName     = "solon"
	defaultVaultAudience       = "vault"
	peisistratosPolicyName     = "peisistratos"
	peisistratosVaultAudience  = "peisistratos"
	gcp                        = "gcp"
	defaultConfigMapAnnotation = "unsealprovider.peisistratos"
)

var (
	//go:embed hcl/policies
	embedPolicies embed.FS
)

type UnsealConfig interface{}

type GCPConfig struct {
	KeyRing   string
	CryptoKey string
	Location  string
}

type AzureConfig struct {
}

func (p *PeisistratosHandler) InitVault(ctx context.Context) error {
	logging.Debug("init for vault in container start")

	status, err := p.Vault.Status(ctx)
	if err != nil {
		return err
	}

	jsonStatus, err := json.MarshalIndent(status, "", "\t")
	if err != nil {
		return err
	}
	logging.Debug(fmt.Sprintf("vault status: %s", jsonStatus))

	if !status.Initialized {
		if err := p.bootstrapVault(ctx); err != nil {
			return err
		}
	} else {
		logging.Debug("vault is already initialized; authenticating for reconciliation")
		if status.Sealed {
			return fmt.Errorf("vault is initialized but sealed; configuration cannot be reconciled")
		}
		if err := p.loginForReconciliation(ctx); err != nil {
			return err
		}
	}

	return p.reconcileVault(ctx)
}

func (p *PeisistratosHandler) bootstrapVault(ctx context.Context) error {
	logging.Debug("vault is not initialized so first step is initializing it")

	nodes, err := p.getVaultPodNodes()
	if err != nil {
		return err
	}

	for _, node := range nodes {
		logging.Debug(fmt.Sprintf("vault pod node: %s", node))
	}

	var init *api.InitResponse

	err = p.determineUnsealMethod()
	if err != nil {
		logging.Debug("could not determine unseal method")
	}

	if p.UnsealMethod != "" {
		logging.Info("initializing vault with auto unseal")
		init, err = p.Vault.InitializeAutoUnseal(ctx, 1, 1)
		if err != nil {
			return err
		}
	} else {
		logging.Info("initializing vault without auto unseal")
		init, err = p.Vault.Initialize(ctx, p.Shares, p.Threshold)
		if err != nil {
			return err
		}

		logging.Debug(fmt.Sprintf("vault is initialized with the following shares: %s", init.Keys))
	}

	logging.Debug(fmt.Sprintf("vault is initialized root token: %s", init.RootToken))

	if len(nodes) > 1 {
		err := p.haFlow(ctx, nodes, init)
		if err != nil {
			return err
		}

	} else {
		err := p.unsealVault(ctx, init)
		if err != nil {
			return err
		}
	}

	if err := p.Vault.LoginWithRootToken(init.RootToken); err != nil {
		return err
	}

	if err := p.Vault.EnableKVSecretsEngine(ctx, "", "configs"); err != nil {
		return err
	}

	return nil
}

func (p *PeisistratosHandler) reconcileVault(ctx context.Context) error {
	if err := p.writeEmbeddedPolicies(ctx); err != nil {
		return err
	}

	return p.configureKubernetesAuth(ctx)
}

func (p *PeisistratosHandler) writeEmbeddedPolicies(ctx context.Context) error {
	files, err := embedPolicies.ReadDir("hcl/policies")
	if err != nil {
		return err
	}

	for _, file := range files {
		if file.IsDir() {
			continue // Skip directories
		}

		// Read the content of the HCL file
		content, err := embedPolicies.ReadFile(fmt.Sprintf("hcl/policies/%s", file.Name()))
		if err != nil {
			logging.Debug(fmt.Sprintf("Error reading file %s: %v\n", file.Name(), err))
			continue
		}

		policyName := strings.TrimSuffix(file.Name(), filepath.Ext(file.Name()))
		if err := p.Vault.WritePolicy(ctx, policyName, content); err != nil {
			return err
		}
	}
	return nil
}

func (p *PeisistratosHandler) configureKubernetesAuth(ctx context.Context) error {
	kubeHostAddress := "https://kubernetes.default.svc"
	serviceAccountName := fmt.Sprintf("%s-access-sa", defaultAdminPolicyName)
	vaultClient, ok := p.Vault.(*diogenes.Vault)
	if !ok || vaultClient.Connection == nil {
		return fmt.Errorf("Vault client does not expose a connection for auth reconciliation")
	}

	auths, err := vaultClient.Connection.Sys().ListAuthWithContext(ctx)
	if err != nil {
		return err
	}
	if _, exists := auths["kubernetes/"]; !exists {
		if err := vaultClient.Connection.Sys().EnableAuthWithOptionsWithContext(ctx, "kubernetes", &api.EnableAuthOptions{
			Type:        "kubernetes",
			Description: "Kubernetes authentication",
		}); err != nil {
			return err
		}
	}

	if _, err := vaultClient.Connection.Logical().WriteWithContext(ctx, "auth/kubernetes/config", map[string]interface{}{
		"kubernetes_host":        kubeHostAddress,
		"disable_iss_validation": true,
	}); err != nil {
		return err
	}

	roles := []struct {
		name     string
		audience string
	}{
		{name: defaultAdminPolicyName, audience: defaultVaultAudience},
		{name: peisistratosPolicyName, audience: peisistratosVaultAudience},
	}
	for _, role := range roles {
		if _, err := vaultClient.Connection.Logical().WriteWithContext(ctx, "auth/kubernetes/role/"+role.name, map[string]interface{}{
			"bound_service_account_names":      []string{serviceAccountName},
			"bound_service_account_namespaces": []string{p.Namespace},
			"policies":                         []string{role.name},
			"audience":                         role.audience,
		}); err != nil {
			return err
		}
	}

	return nil
}

func (p *PeisistratosHandler) determineUnsealMethod() error {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	cfgMaps, err := p.Kube.CoreV1().ConfigMaps(p.Namespace).List(ctx, metav1.ListOptions{})

	if err != nil {
		return err
	}

	for _, cfgMap := range cfgMaps.Items {
		if value, exists := cfgMap.Annotations[defaultConfigMapAnnotation]; exists {
			p.UnsealMethod = value
			return nil
		}
	}

	return nil
}

func (p *PeisistratosHandler) haFlow(ctx context.Context, nodes []string, init *api.InitResponse) error {
	rootPath := plato.StringFromEnv(plato.EnvRootTlSDir, plato.DefaultTLSFileLocation)
	secretPath := filepath.Join(rootPath, "vault")
	if p.Env == "LOCAL" {
		secretPath = "/tmp"
	}

	ca := fmt.Sprintf("%s/vault.ca", secretPath)
	cert := fmt.Sprintf("%s/vault.crt", secretPath)
	key := fmt.Sprintf("%s/vault.key", secretPath)

	_, err := p.Vault.Unseal(ctx, init.Keys)
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
	leader, _ := p.Vault.Leader(ctx)
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

			tlsConfig := diogenes.CreateTLSConfig(ca, cert, key, secretPath)
			tempClient, err := diogenes.NewVaultClient(vaultPodDns, init.RootToken, tlsConfig)
			if err != nil {
				return err
			}

			if server != PrimaryNode {
				err = tempClient.LoginWithRootToken(init.RootToken)
				if err != nil {
					return err
				}

				raft, err := tempClient.RaftJoin(ctx, primaryAddress, readOutCert, readOutKey, readOutCa)
				if err != nil {
					return err
				}

				logging.Debug(fmt.Sprintf("raft joined: %v", raft.Joined))
			}

			err = p.unsealVault(ctx, init)
			if err != nil {
				return err
			}

		} else {
			vaultPodDns := fmt.Sprintf("https://%s.%s.svc.cluster.local:%v", server, p.Namespace, 8200)

			tlsConfig := diogenes.CreateTLSConfig(ca, cert, key, secretPath)
			tempClient, err := diogenes.NewVaultClient(vaultPodDns, init.RootToken, tlsConfig)
			if err != nil {
				return err
			}

			if server != PrimaryNode {
				err = tempClient.LoginWithRootToken(init.RootToken)
				if err != nil {
					return err
				}

				raft, err := tempClient.RaftJoin(ctx, primaryAddress, readOutCert, readOutKey, readOutCa)
				if err != nil {
					return err
				}

				logging.Debug(fmt.Sprintf("raft joined: %v", raft.Joined))
			}

			err = p.unsealVault(ctx, init)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (p *PeisistratosHandler) unsealVault(ctx context.Context, init *api.InitResponse) error {
	var unseal bool
	var err error
	switch p.UnsealMethod {
	case gcp:
		config := createUnsealConfig(gcp).(GCPConfig)
		unseal, err = p.Vault.AutoUnsealGCP(ctx, config.KeyRing, config.CryptoKey, config.Location, init.RecoveryKeys)
		if err != nil {
			return err
		}

	default:
		unseal, err = p.Vault.Unseal(ctx, init.Keys)
		if err != nil {
			return err
		}
	}

	logging.Debug(fmt.Sprintf("unsealed vault : %v", unseal))
	return nil
}

func (p *PeisistratosHandler) getVaultPodNodes() ([]string, error) {
	var nodes []string

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()
	workload, err := p.Kube.AppsV1().StatefulSets(p.Namespace).List(ctx, metav1.ListOptions{})
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

			labelString = labelString[:len(labelString)-2]

			pods, _ := p.Kube.CoreV1().Pods(p.Namespace).List(ctx, metav1.ListOptions{
				LabelSelector: labelString,
			})

			for _, pod := range pods.Items {
				nodes = append(nodes, pod.Name)
			}
		}
	}

	return nodes, nil
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
