package lawgiver

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/odysseia-greek/agora/aristoteles"
	elasticmodels "github.com/odysseia-greek/agora/aristoteles/models"
	"github.com/odysseia-greek/agora/diogenes"
	plato "github.com/odysseia-greek/agora/plato/config"
	"github.com/odysseia-greek/agora/plato/generator"
	"github.com/odysseia-greek/agora/plato/logging"
	"github.com/odysseia-greek/agora/plato/middleware"
	"github.com/odysseia-greek/agora/plato/models"
	kubernetes "github.com/odysseia-greek/agora/thales"
	pb "github.com/odysseia-greek/attike/aristophanes/proto"
	delphi "github.com/odysseia-greek/delphi/solon/models"
	"net/http"
	"strings"
	"time"
)

type SolonHandler struct {
	Vault            diogenes.Client
	Elastic          aristoteles.Client
	ElasticCert      []byte
	Kube             *kubernetes.KubeClient
	Namespace        string
	AccessAnnotation string
	RoleAnnotation   string
	TLSEnabled       bool
	Streamer         pb.TraceService_ChorusClient
	Cancel           context.CancelFunc
}

func (s *SolonHandler) Health(w http.ResponseWriter, req *http.Request) {
	requestId := req.Header.Get(plato.HeaderKey)
	w.Header().Set(plato.HeaderKey, requestId)

	vaultHealth, _ := s.Vault.Health()

	elasticHealth := s.Elastic.Health().Info()
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
	pod, err := s.verifyRequestOriginIP(req.RemoteAddr)
	if err != nil {
		logging.Error(err.Error())
		e := models.ValidationError{
			ErrorModel: models.ErrorModel{UniqueCode: uuid.New().String()},
			Messages: []models.ValidationMessages{
				{
					Field:   "verifying requestIP with a pod",
					Message: err.Error(),
				},
			},
		}
		middleware.ResponseWithJson(w, e)
		return
	}

	// Define the policy name and Vault path
	policyName := fmt.Sprintf("policy-%s", pod.Name)
	podVaultPath := fmt.Sprintf("configs/data/%s", pod.Name)

	// Define the policy rules
	policyRules := fmt.Sprintf(`
path "%s" {
  capabilities = ["read", "list"]
}
`, podVaultPath)

	err = s.Vault.WritePolicy(policyName, []byte(policyRules))
	if err != nil {
		logging.Error(err.Error())
		e := models.ValidationError{
			ErrorModel: models.ErrorModel{UniqueCode: uuid.New().String()},
			Messages: []models.ValidationMessages{
				{
					Field:   "creating policy",
					Message: err.Error(),
				},
			},
		}
		middleware.ResponseWithJson(w, e)
		return
	}

	token, err := s.Vault.CreateOneTimeToken([]string{policyName})
	if err != nil {
		logging.Error(err.Error())
		e := models.ValidationError{
			ErrorModel: models.ErrorModel{UniqueCode: uuid.New().String()},
			Messages: []models.ValidationMessages{
				{
					Field:   "getting token",
					Message: err.Error(),
				},
			},
		}
		middleware.ResponseWithJson(w, e)
		return
	}

	tokenModel := delphi.TokenResponse{
		Token: token,
	}

	middleware.ResponseWithCustomCode(w, http.StatusOK, tokenModel)
}

func (s *SolonHandler) RegisterService(w http.ResponseWriter, req *http.Request) {
	requestId := req.Header.Get(plato.HeaderKey)
	w.Header().Set(plato.HeaderKey, requestId)

	var creationRequest delphi.SolonCreationRequest
	if err := json.NewDecoder(req.Body).Decode(&creationRequest); err != nil {
		s.handleValidationError(w, "decoding", requestId, err)
		return
	}

	pod, err := s.verifyRequestOriginIP(req.RemoteAddr)
	if err != nil {
		logging.Error(err.Error())
		e := models.ValidationError{
			ErrorModel: models.ErrorModel{UniqueCode: uuid.New().String()},
			Messages: []models.ValidationMessages{
				{
					Field:   "verifying requestIP with a pod",
					Message: err.Error(),
				},
			},
		}
		middleware.ResponseWithJson(w, e)
		return
	}

	if pod.Name != creationRequest.PodName {
		// this error should go to slack or somewhere to see illegal actions
		e := models.ValidationError{
			ErrorModel: models.ErrorModel{UniqueCode: uuid.New().String()},
			Messages: []models.ValidationMessages{
				{
					Field:   "creationRequest.Podname",
					Message: fmt.Sprintf("illegal action detected: %s requested but podname is %s", creationRequest.PodName, pod.Name),
				},
			},
		}
		middleware.ResponseWithJson(w, e)
		return
	}

	validAnnotation := s.areValidAnnotations(pod.Annotations, &creationRequest)
	if !validAnnotation {
		// this error should go to slack or somewhere to see illegal actions
		e := models.ValidationError{
			ErrorModel: models.ErrorModel{UniqueCode: uuid.New().String()},
			Messages: []models.ValidationMessages{
				{
					Field:   "annotations",
					Message: fmt.Sprintf("illegal action detected: %s requested invalid annotations", pod.Name),
				},
			},
		}
		middleware.ResponseWithJson(w, e)
		return
	}

	password, err := generator.RandomPassword(18)
	if err != nil {
		s.handleValidationError(w, "passwordgenerator", requestId, err)
		return
	}

	roleNames := s.generateRoleNames(&creationRequest)

	putUser := elasticmodels.CreateUserRequest{
		Password: password,
		Roles:    roleNames,
		FullName: creationRequest.Username,
		Email:    fmt.Sprintf("%s@odysseia-greek.com", creationRequest.Username),
		Metadata: &elasticmodels.Metadata{Version: 1},
	}

	userCreated, err := s.Elastic.Access().CreateUser(creationRequest.Username, putUser)
	if err != nil {
		s.handleValidationError(w, "createUser", requestId, err)
		return
	}

	logging.Debug(fmt.Sprintf("created new user: %s from pod: %s", creationRequest.Username, pod.Name))
	createRequest := diogenes.CreateSecretRequest{
		Data: diogenes.ElasticConfigVault{
			Username:    creationRequest.Username,
			Password:    password,
			ElasticCERT: string(s.ElasticCert),
		},
	}

	payload, _ := createRequest.Marshal()

	logging.Debug(fmt.Sprintf("created secret: %s", pod.Name))
	secretCreated, err := s.Vault.CreateNewSecret(pod.Name, payload)
	if err != nil {
		s.handleValidationError(w, "createSecret", requestId, err)
		return
	}

	response := models.SolonResponse{SecretCreated: secretCreated, UserCreated: userCreated}
	middleware.ResponseWithCustomCode(w, http.StatusCreated, response)
}

func (s *SolonHandler) handleValidationError(w http.ResponseWriter, field, requestId string, err error) {
	e := models.ValidationError{
		ErrorModel: models.ErrorModel{UniqueCode: requestId},
		Messages: []models.ValidationMessages{
			{
				Field:   field,
				Message: err.Error(),
			},
		},
	}
	middleware.ResponseWithJson(w, e)
}

func (s *SolonHandler) areValidAnnotations(annotations map[string]string, req *delphi.SolonCreationRequest) bool {
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

func (s *SolonHandler) generateRoleNames(req *delphi.SolonCreationRequest) []string {
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
