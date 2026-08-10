package lawgiver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	plato "github.com/odysseia-greek/agora/plato/config"
	"github.com/odysseia-greek/agora/plato/generator"
	"github.com/odysseia-greek/agora/plato/logging"
	"github.com/odysseia-greek/agora/plato/middleware"
	"github.com/odysseia-greek/agora/plato/models"
	arv1 "github.com/odysseia-greek/attike/aristophanes/gen/go/v1"
	limenelastic "github.com/odysseia-greek/delphi/solon/limen/elastic"
	limenkubernetes "github.com/odysseia-greek/delphi/solon/limen/kubernetes"
	limenvault "github.com/odysseia-greek/delphi/solon/limen/vault"
	"github.com/odysseia-greek/delphi/solon/logoi"
)

type SolonHandler struct {
	Vault            *limenvault.Client
	Elastic          *limenelastic.Client
	Kubernetes       *limenkubernetes.Client
	ElasticCert      []byte
	AccessAnnotation string
	RoleAnnotation   string
	TLSEnabled       bool
	Streamer         arv1.TraceService_ChorusClient
	Cancel           context.CancelFunc
}

func NewSolonHandler(cfg *Config) *SolonHandler {
	return &SolonHandler{
		Vault:            cfg.VaultLimen,
		Elastic:          cfg.ElasticLimen,
		Kubernetes:       cfg.KubernetesLimen,
		ElasticCert:      cfg.ElasticCert,
		AccessAnnotation: cfg.AccessAnnotation,
		RoleAnnotation:   cfg.RoleAnnotation,
		TLSEnabled:       cfg.TLSEnabled,
		Streamer:         cfg.Streamer,
		Cancel:           cfg.Cancel,
	}
}

func (s *SolonHandler) Health(w http.ResponseWriter, req *http.Request) {
	requestID := req.Header.Get(plato.HeaderKey)
	w.Header().Set(plato.HeaderKey, requestID)

	vaultHealth, _ := s.Vault.Health(req.Context())
	elasticHealth := s.Elastic.HealthInfo()
	dbHealth := models.DatabaseHealth{
		Healthy:       elasticHealth.Healthy,
		ClusterName:   elasticHealth.ClusterName,
		ServerName:    elasticHealth.ServerName,
		ServerVersion: elasticHealth.ServerVersion,
	}
	healthy := models.Health{
		Healthy:  vaultHealth,
		Time:     time.Now().String(),
		Database: dbHealth,
	}
	middleware.ResponseWithJson(w, healthy)
}

func (s *SolonHandler) CreateOneTimeToken(w http.ResponseWriter, req *http.Request) {
	pod, err := s.Kubernetes.VerifyRequestOriginIP(req.RemoteAddr)
	if err != nil {
		logging.Error(err.Error())
		e := models.ValidationError{
			ErrorModel: models.ErrorModel{UniqueCode: uuid.New().String()},
			Messages:   []models.ValidationMessages{{Field: "verifying requestIP with a pod", Message: err.Error()}},
		}
		middleware.ResponseWithJson(w, e)
		return
	}

	if pod == nil {
		e := models.ValidationError{
			ErrorModel: models.ErrorModel{UniqueCode: uuid.New().String()},
			Messages:   []models.ValidationMessages{{Field: "listPods", Message: "no pods could be found"}},
		}
		middleware.ResponseWithJson(w, e)
		return
	}

	policyName := fmt.Sprintf("policy-%s", pod.Name)
	podVaultPath := fmt.Sprintf("configs/data/%s", pod.Name)
	policyRules := fmt.Sprintf("\npath \"%s\" {\n  capabilities = [\"read\", \"list\"]\n}\n", podVaultPath)

	err = s.Vault.WritePolicy(req.Context(), policyName, []byte(policyRules))
	if err != nil {
		logging.Error(err.Error())
		e := models.ValidationError{
			ErrorModel: models.ErrorModel{UniqueCode: uuid.New().String()},
			Messages:   []models.ValidationMessages{{Field: "creating policy", Message: err.Error()}},
		}
		middleware.ResponseWithJson(w, e)
		return
	}

	token, err := s.Vault.CreateOneTimeToken(req.Context(), []string{policyName})
	if err != nil {
		logging.Error(err.Error())
		e := models.ValidationError{
			ErrorModel: models.ErrorModel{UniqueCode: uuid.New().String()},
			Messages:   []models.ValidationMessages{{Field: "getting token", Message: err.Error()}},
		}
		middleware.ResponseWithJson(w, e)
		return
	}

	middleware.ResponseWithCustomCode(w, http.StatusOK, logoi.TokenResponse{Token: token})
}

func (s *SolonHandler) RegisterService(w http.ResponseWriter, req *http.Request) {
	requestID := req.Header.Get(plato.HeaderKey)
	w.Header().Set(plato.HeaderKey, requestID)

	var creationRequest logoi.SolonCreationRequest
	if err := json.NewDecoder(req.Body).Decode(&creationRequest); err != nil {
		s.handleValidationError(w, "decoding", requestID, err)
		return
	}

	pod, err := s.Kubernetes.VerifyRequestOriginIP(req.RemoteAddr)
	if err != nil {
		logging.Error(err.Error())
		e := models.ValidationError{
			ErrorModel: models.ErrorModel{UniqueCode: uuid.New().String()},
			Messages:   []models.ValidationMessages{{Field: "verifying requestIP with a pod", Message: err.Error()}},
		}
		middleware.ResponseWithJson(w, e)
		return
	}

	if pod.Name != creationRequest.PodName {
		e := models.ValidationError{
			ErrorModel: models.ErrorModel{UniqueCode: uuid.New().String()},
			Messages: []models.ValidationMessages{{
				Field:   "creationRequest.Podname",
				Message: fmt.Sprintf("illegal action detected: %s requested but podname is %s", creationRequest.PodName, pod.Name),
			}},
		}
		middleware.ResponseWithJson(w, e)
		return
	}

	if !s.areValidAnnotations(pod.Annotations, &creationRequest) {
		e := models.ValidationError{
			ErrorModel: models.ErrorModel{UniqueCode: uuid.New().String()},
			Messages: []models.ValidationMessages{{
				Field:   "annotations",
				Message: fmt.Sprintf("illegal action detected: %s requested invalid annotations", pod.Name),
			}},
		}
		middleware.ResponseWithJson(w, e)
		return
	}

	password, err := generator.RandomPassword(18)
	if err != nil {
		s.handleValidationError(w, "passwordgenerator", requestID, err)
		return
	}

	roleNames := generateRoleNames(&creationRequest)
	userCreated, err := s.Elastic.CreateUser(creationRequest.Username, password, roleNames)
	if err != nil {
		s.handleValidationError(w, "createUser", requestID, err)
		return
	}

	logging.Debug(fmt.Sprintf("created new user: %s from pod: %s", creationRequest.Username, pod.Name))
	logging.Debug(fmt.Sprintf("created secret: %s", pod.Name))
	secretCreated, err := s.Vault.CreateElasticSecret(req.Context(),
		pod.Name,
		creationRequest.Username,
		password,
		string(s.ElasticCert),
	)
	if err != nil {
		s.handleValidationError(w, "createSecret", requestID, err)
		return
	}

	middleware.ResponseWithCustomCode(w, http.StatusCreated, models.SolonResponse{
		SecretCreated: secretCreated,
		UserCreated:   userCreated,
	})
}

func (s *SolonHandler) handleValidationError(w http.ResponseWriter, field, requestID string, err error) {
	e := models.ValidationError{
		ErrorModel: models.ErrorModel{UniqueCode: requestID},
		Messages:   []models.ValidationMessages{{Field: field, Message: err.Error()}},
	}
	middleware.ResponseWithJson(w, e)
}

func (s *SolonHandler) areValidAnnotations(annotations map[string]string, req *logoi.SolonCreationRequest) bool {
	var validAccess bool
	var validRole bool

	for key, value := range annotations {
		if key == s.AccessAnnotation {
			splittedValues := strings.Split(value, ";")
			for _, a := range req.Access {
				if sliceContains(splittedValues, a) {
					validAccess = true
					break
				}
			}
		} else if key == s.RoleAnnotation && value == req.Role {
			validRole = true
		}
	}

	return validAccess && validRole
}

func generateRoleNames(req *logoi.SolonCreationRequest) []string {
	var roleNames []string
	for _, a := range req.Access {
		roleName := fmt.Sprintf("%s_%s", a, req.Role)
		roleNames = append(roleNames, roleName)
	}
	return roleNames
}

func sliceContains(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}
