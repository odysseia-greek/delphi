package dokime

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	uuid2 "github.com/google/uuid"
	elastic "github.com/odysseia-greek/agora/aristoteles"
	vault "github.com/odysseia-greek/agora/diogenes"
	"github.com/odysseia-greek/agora/plato/config"
	"github.com/odysseia-greek/agora/plato/models"
	"github.com/odysseia-greek/agora/plato/service"
	kubernetes "github.com/odysseia-greek/agora/thales"
	"github.com/odysseia-greek/delphi/solon/lawgiver"
	limenelastic "github.com/odysseia-greek/delphi/solon/limen/elastic"
	limenkubernetes "github.com/odysseia-greek/delphi/solon/limen/kubernetes"
	limenvault "github.com/odysseia-greek/delphi/solon/limen/vault"
	"github.com/odysseia-greek/delphi/solon/logoi"
	"github.com/odysseia-greek/delphi/solon/stoa"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestHealth(t *testing.T) {
	t.Run("HappyPath", func(t *testing.T) {
		fixtureFile := "info"
		mockCode := 200
		mockElasticClient, err := elastic.NewMockClient(fixtureFile, mockCode)
		assert.Nil(t, err)
		vaultFixtures := []string{"health"}
		mockVaultClient, err := vault.CreateMockVaultClient(vaultFixtures, mockCode)
		assert.Nil(t, err)

		testConfig := &lawgiver.SolonHandler{
			Elastic: limenelastic.NewClient(mockElasticClient),
			Vault:   limenvault.NewClient(mockVaultClient),
		}

		router := stoa.InitRoutes(testConfig)
		response := performGetRequest(router, "/solon/v1/health")

		var healthModel models.Health
		err = json.NewDecoder(response.Body).Decode(&healthModel)
		assert.Nil(t, err)
		assert.Equal(t, http.StatusOK, response.Code)
		assert.True(t, healthModel.Healthy)
	})
}

func TestRegister(t *testing.T) {
	access := "everywhere"
	creationRequest := logoi.SolonCreationRequest{
		Role:     "theonethatquestions",
		Access:   []string{access},
		PodName:  "somepodname-122",
		Username: "sokrates",
	}

	var namespaces logoi.Namespaces
	namespaces.SolonNamespace = config.StringFromEnv(config.EnvNamespace, config.DefaultNamespace)

	t.Run("HappyPath", func(t *testing.T) {
		fixtureFile := "createUser"
		mockCode := 200
		mockElasticClient, err := elastic.NewMockClient(fixtureFile, mockCode)
		assert.Nil(t, err)
		vaultFixtures := []string{"createSecret"}
		mockVaultClient, err := vault.CreateMockVaultClient(vaultFixtures, mockCode)
		assert.Nil(t, err)
		mockKube := kubernetes.NewFakeKubeClient()

		testConfig := &lawgiver.SolonHandler{
			Elastic:          limenelastic.NewClient(mockElasticClient),
			Vault:            limenvault.NewClient(mockVaultClient),
			Kubernetes:       limenkubernetes.NewClient(mockKube, namespaces, nil),
			AccessAnnotation: "odysseia-greek/access",
			RoleAnnotation:   "odysseia-greek/role",
		}

		err = createPodForTest(creationRequest.PodName, namespaces.SolonNamespace, access, creationRequest.Role, mockKube)
		assert.Nil(t, err)

		jsonBody, err := creationRequest.Marshal()
		assert.Nil(t, err)
		bodyInBytes := bytes.NewReader(jsonBody)

		router := stoa.InitRoutes(testConfig)
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
		mockKube := kubernetes.NewFakeKubeClient()
		assert.Nil(t, err)

		testConfig := &lawgiver.SolonHandler{
			Elastic:          limenelastic.NewClient(mockElasticClient),
			Vault:            limenvault.NewClient(nil),
			Kubernetes:       limenkubernetes.NewClient(mockKube, namespaces, nil),
			AccessAnnotation: "odysseia-greek/access",
			RoleAnnotation:   "odysseia-greek/role",
		}

		differentRole := "nottheroleyouarelookingfor"

		err = createPodForTest(creationRequest.PodName, namespaces.SolonNamespace, access, differentRole, mockKube)
		assert.Nil(t, err)

		jsonBody, err := creationRequest.Marshal()
		assert.Nil(t, err)
		bodyInBytes := bytes.NewReader(jsonBody)

		router := stoa.InitRoutes(testConfig)
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
		mockKube := kubernetes.NewFakeKubeClient()

		testConfig := &lawgiver.SolonHandler{
			Elastic:          limenelastic.NewClient(mockElasticClient),
			Vault:            limenvault.NewClient(nil),
			Kubernetes:       limenkubernetes.NewClient(mockKube, namespaces, nil),
			AccessAnnotation: "odysseia-greek/access",
			RoleAnnotation:   "odysseia-greek/role",
		}

		differentAccess := "nottheroleyouarelookingfor"

		err = createPodForTest(creationRequest.PodName, namespaces.SolonNamespace, differentAccess, creationRequest.Role, mockKube)
		assert.Nil(t, err)

		jsonBody, err := creationRequest.Marshal()
		assert.Nil(t, err)
		bodyInBytes := bytes.NewReader(jsonBody)

		router := stoa.InitRoutes(testConfig)
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
		mockKube := kubernetes.NewFakeKubeClient()

		testConfig := &lawgiver.SolonHandler{
			Elastic:          limenelastic.NewClient(mockElasticClient),
			Vault:            limenvault.NewClient(nil),
			Kubernetes:       limenkubernetes.NewClient(mockKube, namespaces, nil),
			AccessAnnotation: "odysseia-greek/access",
			RoleAnnotation:   "odysseia-greek/role",
		}

		err = createPodForTest(creationRequest.PodName, namespaces.SolonNamespace, access, creationRequest.Role, mockKube)
		assert.Nil(t, err)

		jsonBody, err := creationRequest.Marshal()
		assert.Nil(t, err)
		bodyInBytes := bytes.NewReader(jsonBody)

		router := stoa.InitRoutes(testConfig)
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
		mockKube := kubernetes.NewFakeKubeClient()
		assert.Nil(t, err)
		vaultClient, err := vault.NewVaultClient("localhost:239riwefj", "token", nil)
		assert.Nil(t, err)

		testConfig := &lawgiver.SolonHandler{
			Elastic:          limenelastic.NewClient(mockElasticClient),
			Kubernetes:       limenkubernetes.NewClient(mockKube, namespaces, nil),
			Vault:            limenvault.NewClient(vaultClient),
			AccessAnnotation: "odysseia-greek/access",
			RoleAnnotation:   "odysseia-greek/role",
		}

		err = createPodForTest(creationRequest.PodName, namespaces.SolonNamespace, access, creationRequest.Role, mockKube)
		assert.Nil(t, err)

		jsonBody, err := creationRequest.Marshal()
		assert.Nil(t, err)
		bodyInBytes := bytes.NewReader(jsonBody)

		router := stoa.InitRoutes(testConfig)
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
	uuid := uuid2.New().String()
	req, _ := http.NewRequest("GET", path, nil)
	req.Header.Set(service.HeaderKey, uuid)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func performPostRequest(r http.Handler, path string, body io.Reader) *httptest.ResponseRecorder {
	uuid := uuid2.New().String()
	req, _ := http.NewRequest("POST", path, body)
	req.Header.Set(service.HeaderKey, uuid)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func createPodForTest(name, ns, access, role string, client *kubernetes.KubeClient) error {
	pod := kubernetes.TestPodObject(name, ns, access, role)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()
	_, err := client.CoreV1().Pods(ns).Create(ctx, pod, metav1.CreateOptions{})
	return err
}
