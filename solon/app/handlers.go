package app

import (
	"encoding/json"
	"fmt"
	elasticmodels "github.com/odysseia-greek/agora/aristoteles/models"
	"github.com/odysseia-greek/agora/diogenes"
	plato "github.com/odysseia-greek/agora/plato/config"
	"github.com/odysseia-greek/agora/plato/generator"
	"github.com/odysseia-greek/agora/plato/logging"
	"github.com/odysseia-greek/agora/plato/middleware"
	"github.com/odysseia-greek/agora/plato/models"
	"github.com/odysseia-greek/delphi/solon/config"
	delphi "github.com/odysseia-greek/delphi/solon/models"
	"net/http"
	"strings"
	"time"
)

type SolonHandler struct {
	Config *config.Config
}

// PingPong pongs the ping
func (s *SolonHandler) PingPong(w http.ResponseWriter, req *http.Request) {
	// swagger:route GET /ping status ping
	//
	// Checks if api is reachable
	//
	//	Consumes:
	//	- application/json
	//
	//	Produces:
	//	- application/json
	//
	//	Schemes: http, https
	//
	//	Responses:
	//	  200: ResultModel
	pingPong := models.ResultModel{Result: "pong"}
	middleware.ResponseWithJson(w, pingPong)
}

func (s *SolonHandler) Health(w http.ResponseWriter, req *http.Request) {
	// swagger:route GET /health status health
	//
	// Checks if api is healthy
	//
	//	Consumes:
	//	- application/json
	//
	//	Produces:
	//	- application/json
	//
	//	Schemes: http, https
	//
	//	Responses:
	//	  200: Health
	//	  502: Health
	requestId := req.Header.Get(plato.HeaderKey)
	w.Header().Set(plato.HeaderKey, requestId)

	vaultHealth, _ := s.Config.Vault.Health()

	elasticHealth := s.Config.Elastic.Health().Info()
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
	// swagger:route GET /token service createToken
	//
	// Creates a one time token for vault
	//
	//	Consumes:
	//	- application/json
	//
	//	Produces:
	//	- application/json
	//
	//	Schemes: http, https
	//
	//	Responses:
	//	  200: TokenResponse
	//    400: ValidationError
	//	  405: MethodError
	requestId := req.Header.Get(plato.HeaderKey)
	w.Header().Set(plato.HeaderKey, requestId)
	//validate podname as registered?
	policy := []string{"ptolemaios"}
	token, err := s.Config.Vault.CreateOneTimeToken(policy)
	if err != nil {
		logging.Error(err.Error())
		e := models.ValidationError{
			ErrorModel: models.ErrorModel{UniqueCode: requestId},
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

// swagger:parameters registerService
type registerServiceParameters struct {
	// in:body
	Application delphi.SolonCreationRequest
}

// RegisterService registers and creates a new user in Elastic which will be stored in vault.
//
// swagger:route POST /register service registerService
//
// Registers and creates a new user in Elastic which will be stored in vault.
//
// Consumes:
// - application/json
//
// Produces:
// - application/json
// Schemes: http, https
//
// Responses:
//
//	200: SolonResponse
//	400: ValidationError
//	405: MethodError
func (s *SolonHandler) RegisterService(w http.ResponseWriter, req *http.Request) {
	requestId := req.Header.Get(plato.HeaderKey)
	w.Header().Set(plato.HeaderKey, requestId)

	var creationRequest delphi.SolonCreationRequest
	if err := json.NewDecoder(req.Body).Decode(&creationRequest); err != nil {
		s.handleValidationError(w, "decoding", requestId, err)
		return
	}

	password, err := generator.RandomPassword(18)
	if err != nil {
		s.handleValidationError(w, "passwordgenerator", requestId, err)
		return
	}

	pod, err := s.Config.Kube.Workload().GetPodByName(s.Config.Namespace, creationRequest.PodName)
	if err != nil || !s.isValidAnnotations(pod.Annotations, &creationRequest) {
		s.handleValidationError(w, "annotations", requestId, fmt.Errorf("annotations requested and found on pod %s did not match", creationRequest.PodName))
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

	userCreated, err := s.Config.Elastic.Access().CreateUser(creationRequest.Username, putUser)
	if err != nil {
		s.handleValidationError(w, "createUser", requestId, err)
		return
	}

	createRequest := diogenes.CreateSecretRequest{
		Data: diogenes.ElasticConfigVault{
			Username:    creationRequest.Username,
			Password:    password,
			ElasticCERT: string(s.Config.ElasticCert),
		},
	}

	payload, _ := createRequest.Marshal()

	secretCreated, err := s.Config.Vault.CreateNewSecret(creationRequest.Username, payload)
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

func (s *SolonHandler) isValidAnnotations(annotations map[string]string, req *delphi.SolonCreationRequest) bool {
	var validAccess bool
	var validRole bool

	for key, value := range annotations {
		if key == s.Config.AccessAnnotation {
			splittedValues := strings.Split(value, ";")
			for _, a := range req.Access {
				if sliceContains(splittedValues, a) {
					validAccess = true
					break
				}
			}
		} else if key == s.Config.RoleAnnotation && value == req.Role {
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
