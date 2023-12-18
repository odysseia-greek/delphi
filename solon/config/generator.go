package config

import (
	"context"
	"fmt"
	"github.com/odysseia-greek/agora/aristoteles"
	elasticmodels "github.com/odysseia-greek/agora/aristoteles/models"
	"github.com/odysseia-greek/agora/diogenes"
	"github.com/odysseia-greek/agora/plato/config"
	"github.com/odysseia-greek/agora/plato/generator"
	"github.com/odysseia-greek/agora/plato/logging"
	kubernetes "github.com/odysseia-greek/agora/thales"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
	"time"
)

const testOverWrite string = "TEST_OVERWRITE"

type Config struct {
	Vault            diogenes.Client
	Elastic          aristoteles.Client
	ElasticCert      []byte
	Kube             *kubernetes.KubeClient
	Namespace        string
	AccessAnnotation string
	RoleAnnotation   string
	TLSEnabled       bool
}

func CreateNewConfig(env string) (*Config, error) {
	healthCheck := true
	outOfClusterKube := false
	debugMode := false
	if env == "DEVELOPMENT" {
		healthCheck = false
		outOfClusterKube = true
		debugMode = true
	}

	testOverWrite := config.BoolFromEnv(testOverWrite)
	tls := config.BoolFromEnv(config.EnvTlSKey)

	vault, err := diogenes.CreateVaultClient(env, healthCheck, debugMode)
	if err != nil {
		return nil, err
	}

	kube, err := kubernetes.CreateKubeClient(outOfClusterKube)
	if err != nil {
		return nil, err
	}

	var cfg elasticmodels.Config
	var cert string

	cfg = aristoteles.ElasticConfig(env, testOverWrite, tls)
	cert = cfg.ElasticCERT

	elastic, err := aristoteles.NewClient(cfg)
	if err != nil {
		return nil, err
	}

	if healthCheck {
		err := aristoteles.HealthCheck(elastic)
		if err != nil {
			return nil, err
		}
	}

	ns := config.StringFromEnv(config.EnvNamespace, config.DefaultNamespace)

	return &Config{
		Vault:            vault,
		Elastic:          elastic,
		ElasticCert:      []byte(cert),
		Kube:             kube,
		Namespace:        ns,
		AccessAnnotation: config.DefaultAccessAnnotation,
		RoleAnnotation:   config.DefaultRoleAnnotation,
		TLSEnabled:       tls,
	}, nil
}

func (s *Config) CreateTracingUser(update bool) error {
	password, err := generator.RandomPassword(24)
	if err != nil {
		return err
	}

	secretName := fmt.Sprintf("%s-elastic", config.DefaultTracingName)
	secretData := map[string][]byte{
		"user":     []byte(config.DefaultTracingName),
		"password": []byte(password),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	secretExists := true
	_, err = s.Kube.CoreV1().Secrets(s.Namespace).Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			secretExists = false
		}
	}

	if secretExists {
		logging.Info(fmt.Sprintf("secret %s already exists", secretName))
		if update {
			logging.Info(fmt.Sprintf("secret %s already exists update flag set so updating", secretName))
			updatedSecret := &corev1.Secret{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Secret",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: secretName,
				},
				Immutable:  nil,
				Data:       secretData,
				StringData: nil,
				Type:       corev1.SecretTypeOpaque,
			}
			_, err = s.Kube.CoreV1().Secrets(s.Namespace).Update(ctx, updatedSecret, metav1.UpdateOptions{})
			if err != nil {
				return err
			}
		} else {
			logging.Info(fmt.Sprintf("secret %s already exists so no action required", secretName))
			return nil
		}

	}

	if !secretExists {
		logging.Info(fmt.Sprintf("secret %s does not exist", secretName))

		scr := &corev1.Secret{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Secret",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: secretName,
			},
			Immutable:  nil,
			Data:       secretData,
			StringData: nil,
			Type:       corev1.SecretTypeOpaque,
		}
		_, err = s.Kube.CoreV1().Secrets(s.Namespace).Create(ctx, scr, metav1.CreateOptions{})
		if err != nil {
			return err
		}
	}

	putUser := elasticmodels.CreateUserRequest{
		Password: password,
		Roles:    []string{config.CreatorElasticRole},
		FullName: config.DefaultTracingName,
		Email:    fmt.Sprintf("%s@odysseia-greek.com", config.DefaultTracingName),
		Metadata: &elasticmodels.Metadata{Version: 1},
	}

	userCreated, err := s.Elastic.Access().CreateUser(config.DefaultTracingName, putUser)
	if err != nil {
		return err
	}

	logging.Info(fmt.Sprintf("user %s created: %v in elasticSearch", config.DefaultTracingName, userCreated))

	return nil
}
