package app

import (
	"encoding/json"
	"fmt"
	"github.com/kpango/glg"
	elasticmodels "github.com/odysseia-greek/aristoteles/models"
	"github.com/odysseia-greek/delphi/solon/config"
	delphi "github.com/odysseia-greek/delphi/solon/models"
	"github.com/odysseia-greek/diogenes"
	"github.com/odysseia-greek/plato/generator"
	"github.com/odysseia-greek/plato/middleware"
	"github.com/odysseia-greek/plato/models"
	plato "github.com/odysseia-greek/plato/service"
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
		glg.Error(err)
		e := models.ValidationError{
			ErrorModel: models.ErrorModel{UniqueCode: middleware.CreateUUID()},
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

	glg.Debug(token)

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

func (s *SolonHandler) RegisterService(w http.ResponseWriter, req *http.Request) {
	// swagger:route POST /register service registerService
	//
	// Registers and creates a new user in Elastic which will be stored in vault
	//
	//	Consumes:
	//	- application/json
	//
	//	Produces:
	//	- application/json
	//	Schemes: http, https
	//
	//	Responses:
	//	  200: SolonResponse
	//    400: ValidationError
	//	  405: MethodError
	var creationRequest delphi.SolonCreationRequest
	decoder := json.NewDecoder(req.Body)
	err := decoder.Decode(&creationRequest)

	glg.Debug(creationRequest)
	if err != nil {
		e := models.ValidationError{
			ErrorModel: models.ErrorModel{UniqueCode: middleware.CreateUUID()},
			Messages: []models.ValidationMessages{
				{
					Field:   "decoding",
					Message: err.Error(),
				},
			},
		}
		middleware.ResponseWithJson(w, e)
		return
	}

	password, err := generator.RandomPassword(18)
	if err != nil {
		e := models.ValidationError{
			ErrorModel: models.ErrorModel{UniqueCode: middleware.CreateUUID()},
			Messages: []models.ValidationMessages{
				{
					Field:   "passwordgenerator",
					Message: err.Error(),
				},
			},
		}
		middleware.ResponseWithJson(w, e)
		return
	}

	glg.Debug("checking pod for correct label")
	//check if pod has the correct labels
	pod, err := s.Config.Kube.Workload().GetPodByName(s.Config.Namespace, creationRequest.PodName)
	var validAccess bool
	var validRole bool
	for key, value := range pod.Annotations {
		if key == s.Config.AccessAnnotation {
			splittedValues := strings.Split(value, ";")
			for _, a := range creationRequest.Access {
				contains := sliceContains(splittedValues, a)
				if !contains {
					break
				}
				glg.Debugf("requested %s matched in annotations %s", a, splittedValues)
				validAccess = contains
			}

		} else if key == s.Config.RoleAnnotation {
			if value == creationRequest.Role {
				glg.Debugf("requested %s matched annotation %s", creationRequest.Role, value)
				validRole = true
			}
		} else {
			continue
		}
	}

	if !validAccess || !validRole {
		glg.Debugf("annotations found on pod %s did not match requested", creationRequest.PodName)
		e := models.ValidationError{
			ErrorModel: models.ErrorModel{UniqueCode: middleware.CreateUUID()},
			Messages: []models.ValidationMessages{
				{
					Field:   "annotations",
					Message: fmt.Sprintf("annotations requested and found on pod %s did not match", creationRequest.PodName),
				},
			},
		}
		middleware.ResponseWithJson(w, e)
		return
	}

	glg.Debugf("annotations found on pod %s matched requested", creationRequest.PodName)

	var roleNames []string
	for _, a := range creationRequest.Access {
		roleName := fmt.Sprintf("%s_%s", a, creationRequest.Role)
		glg.Debugf("adding role named: %s to user", roleName)
		roleNames = append(roleNames, roleName)
	}

	putUser := elasticmodels.CreateUserRequest{
		Password: password,
		Roles:    roleNames,
		FullName: creationRequest.Username,
		Email:    fmt.Sprintf("%s@odysseia-greek.com", creationRequest.Username),
		Metadata: &elasticmodels.Metadata{Version: 1},
	}

	var response delphi.SolonResponse
	userCreated, err := s.Config.Elastic.Access().CreateUser(creationRequest.Username, putUser)
	if err != nil {
		glg.Error(err)
		e := models.ValidationError{
			ErrorModel: models.ErrorModel{UniqueCode: middleware.CreateUUID()},
			Messages: []models.ValidationMessages{
				{
					Field:   "createUser",
					Message: err.Error(),
				},
			},
		}
		middleware.ResponseWithJson(w, e)
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
		glg.Error(err)
		e := models.ValidationError{
			ErrorModel: models.ErrorModel{UniqueCode: middleware.CreateUUID()},
			Messages: []models.ValidationMessages{
				{
					Field:   "createSecret",
					Message: err.Error(),
				},
			},
		}
		middleware.ResponseWithJson(w, e)
		return
	}

	glg.Debugf("secret created in vault %t", secretCreated)

	response.Created = userCreated

	middleware.ResponseWithCustomCode(w, http.StatusCreated, response)
	return
}

func sliceContains(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}
