package lawgiver

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
	aristophanes "github.com/odysseia-greek/attike/aristophanes/comedy"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"strings"
	"time"
)

const testOverWrite string = "TEST_OVERWRITE"

func CreateNewConfig(env string, ctx context.Context) (*SolonHandler, error) {
	healthCheck := true
	outOfClusterKube := false
	debugMode := false

	vault, err := diogenes.CreateVaultClient(env, healthCheck, debugMode)
	if err != nil {
		return nil, err
	}

	testOverWrite := config.BoolFromEnv(testOverWrite)
	tls := config.BoolFromEnv(config.EnvTlSKey)

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

	logging.System("creating attike users at startup")
	err = createAttikeUsers(false, kube, elastic, ns)

	tracer, err := aristophanes.NewClientTracer()
	if err != nil {
		logging.Error(err.Error())
	}

	healthy := tracer.WaitForHealthyState()
	if !healthy {
		logging.Debug("tracing service not ready - restarting seems the only option")
		os.Exit(1)
	}

	streamer, err := tracer.Chorus(ctx)
	if err != nil {
		logging.Error(err.Error())
	}

	ctx, cancel := context.WithCancel(ctx)

	return &SolonHandler{
		Vault:            vault,
		Elastic:          elastic,
		ElasticCert:      []byte(cert),
		Kube:             kube,
		Namespace:        ns,
		AccessAnnotation: config.DefaultAccessAnnotation,
		RoleAnnotation:   config.DefaultRoleAnnotation,
		TLSEnabled:       tls,
		Streamer:         streamer,
		Cancel:           cancel,
	}, nil
}

func createAttikeUsers(update bool, kube *kubernetes.KubeClient, elastic aristoteles.Client, namespace string) error {
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
