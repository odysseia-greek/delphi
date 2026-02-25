package limen

import (
	"context"
	"errors"
	"testing"

	"github.com/hashicorp/vault/api"
	"github.com/odysseia-greek/agora/aristoteles"
	elasticmodels "github.com/odysseia-greek/agora/aristoteles/models"
	"github.com/odysseia-greek/agora/diogenes"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type accessSpy struct {
	deleteUserErr   error
	deletedUserName []string
}

func (a *accessSpy) CreateRole(name string, roleRequest elasticmodels.CreateRoleRequest) (bool, error) {
	return false, nil
}

func (a *accessSpy) CreateRoleWithContext(ctx context.Context, name string, roleRequest elasticmodels.CreateRoleRequest) (bool, error) {
	return false, nil
}

func (a *accessSpy) CreateUser(name string, userCreation elasticmodels.CreateUserRequest) (bool, error) {
	return false, nil
}

func (a *accessSpy) CreateUserWithContext(ctx context.Context, name string, userCreation elasticmodels.CreateUserRequest) (bool, error) {
	return false, nil
}

func (a *accessSpy) ListUsers() ([]string, error) {
	return nil, nil
}

func (a *accessSpy) ListUsersWithContext(ctx context.Context) ([]string, error) {
	return nil, nil
}

func (a *accessSpy) DeleteUser(name string) (bool, error) {
	return a.DeleteUserWithContext(context.Background(), name)
}

func (a *accessSpy) DeleteUserWithContext(ctx context.Context, name string) (bool, error) {
	a.deletedUserName = append(a.deletedUserName, name)
	return a.deleteUserErr == nil, a.deleteUserErr
}

type elasticSpy struct {
	access *accessSpy
}

func (e *elasticSpy) Query() aristoteles.Query {
	return nil
}

func (e *elasticSpy) Document() aristoteles.Document {
	return nil
}

func (e *elasticSpy) Index() aristoteles.Index {
	return nil
}

func (e *elasticSpy) Builder() aristoteles.Builder {
	return nil
}

func (e *elasticSpy) Health() aristoteles.Health {
	return nil
}

func (e *elasticSpy) Access() aristoteles.Access {
	return e.access
}

func (e *elasticSpy) Policy() aristoteles.Policy {
	return nil
}

type vaultSpy struct {
	diogenes.Client

	deleteSecretErr  error
	removeSecretErr  error
	deletePolicyErr  error
	deletePolicyResp *api.Secret

	deleteSecretCalls []string
	removeSecretCalls []string
	deletePolicyCalls []string
}

func (v *vaultSpy) DeleteSecret(name string) error {
	v.deleteSecretCalls = append(v.deleteSecretCalls, name)
	return v.deleteSecretErr
}

func (v *vaultSpy) RemoveSecret(name string) error {
	v.removeSecretCalls = append(v.removeSecretCalls, name)
	return v.removeSecretErr
}

func (v *vaultSpy) DeletePolicy(policyName string) (*api.Secret, error) {
	v.deletePolicyCalls = append(v.deletePolicyCalls, policyName)
	return v.deletePolicyResp, v.deletePolicyErr
}

func TestDeleteOrphans_NormalizesHyphenatedPodNameAndCleans(t *testing.T) {
	access := &accessSpy{}
	vault := &vaultSpy{}

	err := DeleteOrphans(&elasticSpy{access: access}, vault, &v1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "my-pod-123"}})
	assert.NoError(t, err)
	assert.Equal(t, []string{"my123"}, access.deletedUserName)
	assert.Equal(t, []string{"my-pod-123"}, vault.deleteSecretCalls)
	assert.Equal(t, []string{"my-pod-123"}, vault.removeSecretCalls)
	assert.Equal(t, []string{"policy-my-pod-123"}, vault.deletePolicyCalls)
}

func TestDeleteOrphans_UsesPodNameWhenNoHyphen(t *testing.T) {
	access := &accessSpy{}
	vault := &vaultSpy{}

	err := DeleteOrphans(&elasticSpy{access: access}, vault, &v1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "standalone"}})
	assert.NoError(t, err)
	assert.Equal(t, []string{"standalone"}, access.deletedUserName)
}

func TestDeleteOrphans_ContinuesCleanupOnClientErrors(t *testing.T) {
	access := &accessSpy{deleteUserErr: errors.New("elastic down")}
	vault := &vaultSpy{
		deleteSecretErr: errors.New("delete failed"),
		removeSecretErr: errors.New("remove failed"),
		deletePolicyErr: errors.New("policy failed"),
	}

	err := DeleteOrphans(&elasticSpy{access: access}, vault, &v1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "my-pod"}})
	assert.NoError(t, err)
	assert.Equal(t, []string{"mypod"}, access.deletedUserName)
	assert.Len(t, vault.deleteSecretCalls, 1)
	assert.Len(t, vault.removeSecretCalls, 1)
	assert.Len(t, vault.deletePolicyCalls, 1)
}
