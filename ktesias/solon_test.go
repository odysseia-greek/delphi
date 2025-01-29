package ktesias

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"github.com/odysseia-greek/agora/plato/config"
	"github.com/odysseia-greek/agora/plato/models"
	"strings"
)

type ElasticResponse struct {
	ElasticCert     string `json:"elasticCert"`
	ElasticUsername string `json:"elasticUsername"`
	ElasticPassword string `json:"elasticPassword"`
}

const (
	TokenContext         string = "tokenContext"
	SecondTokenContext   string = "secondTokenContext"
	ErrorContext         string = "errorContext"
	RegisterContext      string = "registerContext"
	FakePodName          string = "fakePodName"
	ElasticClientContext string = "elasticClientContext"
	ElasticResponseCode  string = "elasticResponseCode"
	VaultConfig          string = "vaultConfig"
)

func (l *OdysseiaFixture) solonReturnsAHealthyResponse() error {
	response, err := l.client.Solon().Health("ktesias-test")
	if err != nil {
		return err
	}

	var healthy *models.Health
	defer response.Body.Close()
	err = json.NewDecoder(response.Body).Decode(&healthy)
	if err != nil {
		return err
	}

	if !healthy.Healthy {
		return fmt.Errorf("expected healthy to be true")
	}

	return nil
}

func (l *OdysseiaFixture) aRequestIsMadeForAOneTimeToken() error {
	response, err := l.client.Solon().OneTimeToken("")
	if err != nil {
		return err
	}

	defer response.Body.Close()

	var tokenModel models.TokenResponse
	err = json.NewDecoder(response.Body).Decode(&tokenModel)
	if err != nil {
		return err
	}

	l.ctx = context.WithValue(l.ctx, TokenContext, tokenModel.Token)
	return nil
}

func (l *OdysseiaFixture) aRequestIsMadeToRegisterTheRunningPodWithIncorrectRoleAndAccessAnnotations() error {
	body := models.SolonCreationRequest{
		Role:     "api",
		Access:   []string{"someindexthatdoesnotexist"},
		PodName:  l.PodName,
		Username: "testpodname",
	}
	response, err := l.client.Solon().Register(body, "")
	defer response.Body.Close()

	var solonResponse models.ValidationError
	err = json.NewDecoder(response.Body).Decode(&solonResponse)
	if err != nil {
		return err
	}

	l.ctx = context.WithValue(l.ctx, ErrorContext, solonResponse)

	return nil
}

func (l *OdysseiaFixture) aRequestIsMadeToRegisterTheRunningPodWithCorrectRoleAndAccessAnnotations() error {
	role := config.StringFromEnv(config.EnvRole, "")
	envAccess := config.SliceFromEnv(config.EnvIndex)

	body := models.SolonCreationRequest{
		Role:     role,
		Access:   envAccess,
		PodName:  l.PodName,
		Username: "testpodname",
	}
	response, err := l.client.Solon().Register(body, "")
	defer response.Body.Close()

	var solonResponse models.SolonResponse
	err = json.NewDecoder(response.Body).Decode(&solonResponse)
	if err != nil {
		return err
	}

	l.ctx = context.WithValue(l.ctx, RegisterContext, solonResponse)

	return nil
}

func (l *OdysseiaFixture) aSuccessfulRegisterShouldBeMade() error {
	register := l.ctx.Value(RegisterContext).(models.SolonResponse)
	if !register.SecretCreated || !register.UserCreated {
		return fmt.Errorf("expected secret and user to be created")
	}

	return nil
}

func (l *OdysseiaFixture) aValidationErrorIsReturnedThatThePodAnnotationsDoNotMatchTheRequestedRoleAndAccess() error {
	registerError := l.ctx.Value(ErrorContext).(models.ValidationError)
	if len(registerError.Messages) == 0 {
		return fmt.Errorf("expected messages to be included but none found")
	}

	if !strings.Contains(registerError.Messages[0].Message, "annotations") || !strings.Contains(registerError.Messages[0].Message, l.PodName) {
		return fmt.Errorf("expected annotations or pod annotations to match")
	}

	return nil
}

func (l *OdysseiaFixture) aOneTimeTokenIsReturned() error {
	token := l.ctx.Value(TokenContext).(string)
	if token == "" {
		return fmt.Errorf("expected token to be non-empty")
	}

	return nil
}

func (l *OdysseiaFixture) aRequestIsMadeToRegisterTheRunningPodWithCorrectRoleAndAccessAnnotationsButAMismatchedPodname() error {
	role := config.StringFromEnv(config.EnvRole, "")
	envAccess := config.SliceFromEnv(config.EnvIndex)

	body := models.SolonCreationRequest{
		Role:     role,
		Access:   envAccess,
		PodName:  "thisisnotthepodname",
		Username: "testpodname",
	}
	response, err := l.client.Solon().Register(body, "")
	defer response.Body.Close()

	var solonResponse models.SolonResponse
	err = json.NewDecoder(response.Body).Decode(&solonResponse)
	if err != nil {
		return err
	}

	l.ctx = context.WithValue(l.ctx, FakePodName, body.PodName)
	l.ctx = context.WithValue(l.ctx, RegisterContext, solonResponse)

	return nil
}

func (l *OdysseiaFixture) aRequestIsMadeForASecondOneTimeToken() error {
	response, err := l.client.Solon().OneTimeToken("")
	if err != nil {
		return err
	}

	defer response.Body.Close()

	var tokenModel models.TokenResponse
	err = json.NewDecoder(response.Body).Decode(&tokenModel)
	if err != nil {
		return err
	}

	l.ctx = context.WithValue(l.ctx, SecondTokenContext, tokenModel.Token)
	return nil
}

func (l *OdysseiaFixture) theTokenFromTheActualPodnameIsValid() error {
	var oneTimeToken string
	if token, ok := l.ctx.Value(SecondTokenContext).(string); ok && token != "" {
		oneTimeToken = token
	} else if fallbackToken, ok := l.ctx.Value(TokenContext).(string); ok && fallbackToken != "" {
		oneTimeToken = fallbackToken
	} else {
		return fmt.Errorf("both SecondTokenContext and TokenContext are nil or empty")
	}

	l.Vault.SetOnetimeToken(oneTimeToken)
	secret, err := l.Vault.GetSecret(l.PodName)
	if err != nil {
		return err
	}

	for key, value := range secret.Data {
		if key == "data" {

			j, err := json.Marshal(value)
			if err != nil {
				return err
			}

			err = l.validateVaultData(j)
			if err != nil {
				return err
			}

		}
	}

	return nil
}

func (l *OdysseiaFixture) theTokenFromTheMismatchedPodnameIsNotValid() error {
	oneTimeToken := l.ctx.Value(TokenContext).(string)
	fakePodName := l.ctx.Value(FakePodName).(string)

	l.Vault.SetOnetimeToken(oneTimeToken)
	_, err := l.Vault.GetSecret(fakePodName)
	if err == nil {
		return fmt.Errorf("expected pod name to be invalid")
	}

	if !strings.Contains(err.Error(), "permission denied") {
		return fmt.Errorf("expected pod name to be invalid")
	}

	return nil
}

func (l *OdysseiaFixture) theTokensAreNotUsableTwice() error {
	oneTimeToken := l.ctx.Value(SecondTokenContext).(string)
	fakeToken := l.ctx.Value(TokenContext).(string)

	l.Vault.SetOnetimeToken(oneTimeToken)
	_, err := l.Vault.GetSecret(l.PodName)
	if err == nil {
		return fmt.Errorf("expected token to be unusable")
	}

	l.Vault.SetOnetimeToken(fakeToken)
	_, err = l.Vault.GetSecret(l.PodName)
	if err == nil {
		return fmt.Errorf("expected token to be unusable")
	}

	return nil
}

func (l *OdysseiaFixture) validateVaultData(response []byte) error {
	var elasticResponse ElasticResponse
	if err := json.Unmarshal(response, &elasticResponse); err != nil {
		return fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Validate username and password
	if elasticResponse.ElasticUsername == "" {
		return errors.New("missing elastic username")
	}
	if elasticResponse.ElasticPassword == "" {
		return errors.New("missing elastic password")
	}

	// Validate certificate(s)
	certBlocks := strings.Split(elasticResponse.ElasticCert, "-----END CERTIFICATE-----")
	for _, block := range certBlocks {
		block = strings.TrimSpace(block)
		if block == "" {
			continue
		}
		block += "\n-----END CERTIFICATE-----"

		blockBytes := []byte(block)
		_, err := x509.ParseCertificate(blockBytes)
		if err != nil {
			// ParseCertificate requires DER-encoded bytes; we need PEM decoding first
			der, _ := pem.Decode(blockBytes)
			if der == nil {
				return fmt.Errorf("invalid certificate block: %s", block)
			}
			_, err = x509.ParseCertificate(der.Bytes)
			if err != nil {
				return fmt.Errorf("invalid certificate: %w", err)
			}
		}
	}

	return nil
}
