package architect

import (
	"context"
	"fmt"
	"github.com/cilium/cilium/pkg/k8s/client/clientset/versioned"
	"github.com/odysseia-greek/agora/aristoteles"
	elasticmodels "github.com/odysseia-greek/agora/aristoteles/models"
	"github.com/odysseia-greek/agora/plato/config"
	"github.com/odysseia-greek/agora/plato/generator"
	"github.com/odysseia-greek/agora/plato/logging"
	"github.com/odysseia-greek/agora/thales"
	"github.com/odysseia-greek/agora/thales/odysseia"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	ORGANISATION string = "odysseia-greek"
)

type MappingUpdate struct {
	HostName     string
	ClientName   string
	KubeType     string
	SecretName   string
	Validity     int
	IsHostUpdate bool
}

func CreateNewConfig() (*PeriklesHandler, error) {
	kube, err := thales.CreateKubeClient(false)
	if err != nil {
		return nil, err
	}

	ciliumClient, err := versioned.NewForConfig(kube.RestConfig())
	if err != nil {
		return nil, err
	}

	org := []string{
		ORGANISATION,
	}

	mapping, err := odysseia.NewServiceMappingImpl(kube.RestConfig())
	if err != nil {
		return nil, err
	}

	cert, err := config.CreateCertClient(org)
	if err != nil {
		return nil, err
	}

	ns := config.StringFromEnv(config.EnvNamespace, config.DefaultNamespace)
	crd := config.StringFromEnv(config.EnvCrdName, config.DefaultCrdName)
	tlsFiles := config.StringFromEnv(config.EnvTLSFiles, config.DefaultTLSFileLocation)
	l7Mode := config.BoolFromEnv("L7_MODE")

	tls := config.BoolFromEnv(config.EnvTlSKey)
	cfg, err := aristoteles.ElasticConfig(tls)
	if err != nil {
		logging.Error(fmt.Sprintf("failed to create Elastic client operations will be interupted, %s", err.Error()))
	}

	elastic, err := aristoteles.NewClient(cfg)
	if err != nil {
		return nil, err
	}

	err = aristoteles.HealthCheck(elastic)
	if err != nil {
		return nil, err
	}

	logging.System("creating attike users at startup")
	err = createAttikeUsers(false, kube, elastic, ns)

	configMapName := os.Getenv("CONFIGMAP_NAME")
	if configMapName == "" {

	}

	tlsChecker := 1 * time.Hour
	updateMappingTimer := 30 * time.Second

	return &PeriklesHandler{
		Mutex:              sync.Mutex{},
		PendingUpdateTimer: updateMappingTimer,
		TLSCheckTimer:      tlsChecker,
		PendingUpdates:     map[string][]MappingUpdate{},
		Kube:               kube,
		CiliumClient:       ciliumClient,
		Mapping:            mapping,
		Cert:               cert,
		Namespace:          ns,
		CrdName:            crd,
		TLSFiles:           tlsFiles,
		L7Mode:             l7Mode,
		Elastic:            elastic,
		ConfigMapName:      configMapName,
		RuleSet:            make([]CnpRuleSet, 0),
	}, nil
}

func createAttikeUsers(update bool, kube *thales.KubeClient, elastic aristoteles.Client, namespace string) error {
	attikeUsers := []string{config.DefaultTracingName, config.DefaultMetricsName}
	for _, user := range attikeUsers {
		password, err := generator.RandomPassword(24)
		if err != nil {
			return err
		}

		secretName := fmt.Sprintf("%s-elastic", user)
		secretData := map[string][]byte{
			"user":     []byte(user),
			"password": []byte(password),
		}

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
		defer cancel()

		secretExists := true
		_, err = kube.CoreV1().Secrets(namespace).Get(ctx, secretName, metav1.GetOptions{})
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
				_, err = kube.CoreV1().Secrets(namespace).Update(ctx, updatedSecret, metav1.UpdateOptions{})
				if err != nil {
					return err
				}
			} else {
				logging.Info(fmt.Sprintf("secret %s already exists so no action required", secretName))
				continue
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
			_, err = kube.CoreV1().Secrets(namespace).Create(ctx, scr, metav1.CreateOptions{})
			if err != nil {
				return err
			}
		}

		var index string
		switch user {
		case config.DefaultTracingName:
			index = config.TracingElasticIndex
		case config.DefaultMetricsName:
			index = config.MetricsElasticIndex
		}

		roleName := fmt.Sprintf("%s_%s", index, config.CreatorElasticRole)

		putUser := elasticmodels.CreateUserRequest{
			Password: password,
			Roles:    []string{roleName},
			FullName: user,
			Email:    fmt.Sprintf("%s@odysseia-greek.com", user),
			Metadata: &elasticmodels.Metadata{Version: 1},
		}

		userCreated, err := elastic.Access().CreateUser(user, putUser)
		if err != nil {
			return err
		}

		logging.Info(fmt.Sprintf("user %s created: %v in elasticSearch", user, userCreated))
	}

	return nil
}
