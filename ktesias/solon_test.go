package ktesias

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/odysseia-greek/agora/plato/config"
	"github.com/odysseia-greek/agora/plato/models"
	"strings"
)

const (
	TokenContext    string = "tokenContext"
	ErrorContext    string = "errorContext"
	RegisterContext string = "registerContext"
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

func (l *OdysseiaFixture) aRequestIsMadeForAOneTimeTokenWithoutAnnotations() error {
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
