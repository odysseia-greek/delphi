package app

import (
	"bytes"
	"encoding/json"
	elastic "github.com/odysseia-greek/agora/aristoteles"
	vault "github.com/odysseia-greek/agora/diogenes"
	"github.com/odysseia-greek/agora/plato/models"
	kubernetes "github.com/odysseia-greek/agora/thales"
	configs "github.com/odysseia-greek/delphi/solon/config"
	delphi "github.com/odysseia-greek/delphi/solon/models"
	"github.com/stretchr/testify/assert"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPingPongRoute(t *testing.T) {
	testConfig := configs.Config{}
	router := InitRoutes(testConfig)
	expected := "{\"result\":\"pong\"}"

	w := performGetRequest(router, "/solon/v1/ping")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, expected, w.Body.String())
}

func TestHealth(t *testing.T) {
	t.Run("HappyPath", func(t *testing.T) {
		fixtureFile := "info"
		mockCode := 200
		mockElasticClient, err := elastic.NewMockClient(fixtureFile, mockCode)
		assert.Nil(t, err)
		vaultFixtures := []string{"health"}
		mockVaultClient, err := vault.CreateMockVaultClient(vaultFixtures, mockCode)
		assert.Nil(t, err)

		testConfig := configs.Config{
			Elastic: mockElasticClient,
			Vault:   mockVaultClient,
		}

		router := InitRoutes(testConfig)
		response := performGetRequest(router, "/solon/v1/health")

		var healthModel models.Health
		err = json.NewDecoder(response.Body).Decode(&healthModel)
		assert.Nil(t, err)
		assert.Equal(t, http.StatusOK, response.Code)
		assert.True(t, healthModel.Healthy)
	})
}

func TestCreateToken(t *testing.T) {
	t.Run("HappyPath", func(t *testing.T) {
		mockCode := 200
		vaultFixtures := []string{"token"}
		mockVaultClient, err := vault.CreateMockVaultClient(vaultFixtures, mockCode)
		assert.Nil(t, err)

		testConfig := configs.Config{
			Vault: mockVaultClient,
		}

		router := InitRoutes(testConfig)
		response := performGetRequest(router, "/solon/v1/token")

		var token delphi.TokenResponse
		err = json.NewDecoder(response.Body).Decode(&token)
		assert.Nil(t, err)
		assert.Equal(t, http.StatusOK, response.Code)
		assert.Contains(t, token.Token, "s.")
	})

	t.Run("VaultDown", func(t *testing.T) {
		badAddress := "localhost:239riwefj"
		vaultClient, err := vault.NewVaultClient(badAddress, "token", nil)
		assert.Nil(t, err)

		testConfig := configs.Config{
			Vault: vaultClient,
		}

		router := InitRoutes(testConfig)
		response := performGetRequest(router, "/solon/v1/token")

		var sut models.ValidationError
		err = json.NewDecoder(response.Body).Decode(&sut)
		assert.Nil(t, err)
		assert.Equal(t, http.StatusBadRequest, response.Code)
		assert.Contains(t, sut.Messages[0].Field, "token")
		assert.Contains(t, sut.Messages[0].Message, "")
	})
}

func TestRegister(t *testing.T) {
	access := "everywhere"
	creationRequest := delphi.SolonCreationRequest{
		Role:     "theonethatquestions",
		Access:   []string{access},
		PodName:  "somepodname-122",
		Username: "sokrates",
	}

	ns := "test"

	t.Run("HappyPath", func(t *testing.T) {
		fixtureFile := "createUser"
		mockCode := 200
		mockElasticClient, err := elastic.NewMockClient(fixtureFile, mockCode)
		assert.Nil(t, err)
		vaultFixtures := []string{"createSecret"}
		mockVaultClient, err := vault.CreateMockVaultClient(vaultFixtures, mockCode)
		assert.Nil(t, err)
		mockKube, err := kubernetes.FakeKubeClient(ns)
		assert.Nil(t, err)

		testConfig := configs.Config{
			Elastic:          mockElasticClient,
			Vault:            mockVaultClient,
			Kube:             mockKube,
			Namespace:        ns,
			AccessAnnotation: "odysseia-greek/access",
			RoleAnnotation:   "odysseia-greek/role",
		}

		err = kubernetes.CreatePodForTest(creationRequest.PodName, ns, access, creationRequest.Role, mockKube)
		assert.Nil(t, err)

		jsonBody, err := creationRequest.Marshal()
		assert.Nil(t, err)
		bodyInBytes := bytes.NewReader(jsonBody)

		router := InitRoutes(testConfig)
		response := performPostRequest(router, "/solon/v1/register", bodyInBytes)

		var sut models.SolonResponse
		err = json.NewDecoder(response.Body).Decode(&sut)
		assert.Nil(t, err)
		assert.Equal(t, http.StatusCreated, response.Code)
		assert.True(t, sut.SecretCreated)
	})

	t.Run("AnnotationNotOnPodRole", func(t *testing.T) {
		fixtureFile := "createUser"
		mockCode := 200
		mockElasticClient, err := elastic.NewMockClient(fixtureFile, mockCode)
		assert.Nil(t, err)
		mockKube, err := kubernetes.FakeKubeClient(ns)
		assert.Nil(t, err)

		testConfig := configs.Config{
			Elastic:          mockElasticClient,
			Vault:            nil,
			Kube:             mockKube,
			Namespace:        ns,
			AccessAnnotation: "odysseia-greek/access",
			RoleAnnotation:   "odysseia-greek/role",
		}

		differentRole := "nottheroleyouarelookingfor"

		err = kubernetes.CreatePodForTest(creationRequest.PodName, ns, access, differentRole, mockKube)
		assert.Nil(t, err)

		jsonBody, err := creationRequest.Marshal()
		assert.Nil(t, err)
		bodyInBytes := bytes.NewReader(jsonBody)

		router := InitRoutes(testConfig)
		response := performPostRequest(router, "/solon/v1/register", bodyInBytes)

		var sut models.ValidationError
		err = json.NewDecoder(response.Body).Decode(&sut)
		assert.Nil(t, err)
		assert.Equal(t, http.StatusBadRequest, response.Code)
		assert.Equal(t, "annotations", sut.Messages[0].Field)
		assert.Contains(t, sut.Messages[0].Message, creationRequest.PodName)
	})

	t.Run("AnnotationNotOnAccess", func(t *testing.T) {
		fixtureFile := "createUser"
		mockCode := 200
		mockElasticClient, err := elastic.NewMockClient(fixtureFile, mockCode)
		assert.Nil(t, err)
		mockKube, err := kubernetes.FakeKubeClient(ns)
		assert.Nil(t, err)

		testConfig := configs.Config{
			Elastic:          mockElasticClient,
			Vault:            nil,
			Kube:             mockKube,
			Namespace:        ns,
			AccessAnnotation: "odysseia-greek/access",
			RoleAnnotation:   "odysseia-greek/role",
		}

		differentAccess := "nottheroleyouarelookingfor"

		err = kubernetes.CreatePodForTest(creationRequest.PodName, ns, differentAccess, creationRequest.Role, mockKube)
		assert.Nil(t, err)

		jsonBody, err := creationRequest.Marshal()
		assert.Nil(t, err)
		bodyInBytes := bytes.NewReader(jsonBody)

		router := InitRoutes(testConfig)
		response := performPostRequest(router, "/solon/v1/register", bodyInBytes)

		var sut models.ValidationError
		err = json.NewDecoder(response.Body).Decode(&sut)
		assert.Nil(t, err)
		assert.Equal(t, http.StatusBadRequest, response.Code)
		assert.Equal(t, "annotations", sut.Messages[0].Field)
		assert.Contains(t, sut.Messages[0].Message, creationRequest.PodName)
	})

	t.Run("UserCannotBeCreated", func(t *testing.T) {
		fixtureFile := "shardFailure"
		mockCode := 502
		mockElasticClient, err := elastic.NewMockClient(fixtureFile, mockCode)
		assert.Nil(t, err)
		mockKube, err := kubernetes.FakeKubeClient(ns)
		assert.Nil(t, err)

		testConfig := configs.Config{
			Elastic:          mockElasticClient,
			Vault:            nil,
			Kube:             mockKube,
			Namespace:        ns,
			AccessAnnotation: "odysseia-greek/access",
			RoleAnnotation:   "odysseia-greek/role",
		}

		err = kubernetes.CreatePodForTest(creationRequest.PodName, ns, access, creationRequest.Role, mockKube)
		assert.Nil(t, err)

		jsonBody, err := creationRequest.Marshal()
		assert.Nil(t, err)
		bodyInBytes := bytes.NewReader(jsonBody)

		router := InitRoutes(testConfig)
		response := performPostRequest(router, "/solon/v1/register", bodyInBytes)

		var sut models.ValidationError
		err = json.NewDecoder(response.Body).Decode(&sut)
		assert.Nil(t, err)
		assert.Equal(t, http.StatusBadRequest, response.Code)
		assert.Equal(t, "createUser", sut.Messages[0].Field)
		assert.Contains(t, sut.Messages[0].Message, "Bad Gateway")
	})

	t.Run("VaultDown", func(t *testing.T) {
		fixtureFile := "createUser"
		mockCode := 200
		mockElasticClient, err := elastic.NewMockClient(fixtureFile, mockCode)
		assert.Nil(t, err)
		mockKube, err := kubernetes.FakeKubeClient(ns)
		assert.Nil(t, err)
		vaultClient, err := vault.NewVaultClient("localhost:239riwefj", "token", nil)
		assert.Nil(t, err)

		testConfig := configs.Config{
			Elastic:          mockElasticClient,
			Kube:             mockKube,
			Vault:            vaultClient,
			Namespace:        ns,
			AccessAnnotation: "odysseia-greek/access",
			RoleAnnotation:   "odysseia-greek/role",
		}

		err = kubernetes.CreatePodForTest(creationRequest.PodName, ns, access, creationRequest.Role, mockKube)
		assert.Nil(t, err)

		jsonBody, err := creationRequest.Marshal()
		assert.Nil(t, err)
		bodyInBytes := bytes.NewReader(jsonBody)

		router := InitRoutes(testConfig)
		response := performPostRequest(router, "/solon/v1/register", bodyInBytes)

		var sut models.ValidationError
		err = json.NewDecoder(response.Body).Decode(&sut)
		assert.Nil(t, err)
		assert.Equal(t, http.StatusBadRequest, response.Code)
		assert.Equal(t, "createSecret", sut.Messages[0].Field)
		assert.Contains(t, sut.Messages[0].Message, "vault")
	})
}

func performGetRequest(r http.Handler, path string) *httptest.ResponseRecorder {
	req, _ := http.NewRequest("GET", path, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func performPostRequest(r http.Handler, path string, body io.Reader) *httptest.ResponseRecorder {
	req, _ := http.NewRequest("POST", path, body)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}
