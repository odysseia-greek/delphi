package vault

import (
	"context"
	"errors"
	"testing"

	"github.com/hashicorp/vault/api"
	"github.com/odysseia-greek/agora/diogenes"
	"github.com/stretchr/testify/assert"
)

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

func (v *vaultSpy) DeleteSecret(_ context.Context, name string) error {
	v.deleteSecretCalls = append(v.deleteSecretCalls, name)
	return v.deleteSecretErr
}

func (v *vaultSpy) RemoveSecret(_ context.Context, name string) error {
	v.removeSecretCalls = append(v.removeSecretCalls, name)
	return v.removeSecretErr
}

func (v *vaultSpy) DeletePolicy(_ context.Context, policyName string) (*api.Secret, error) {
	v.deletePolicyCalls = append(v.deletePolicyCalls, policyName)
	return v.deletePolicyResp, v.deletePolicyErr
}

func TestDeleteOrphan_CleansVaultResources(t *testing.T) {
	vault := &vaultSpy{}
	client := NewClient(vault)

	err := client.DeleteOrphan(context.Background(), "my-pod-123")
	assert.NoError(t, err)
	assert.Equal(t, []string{"my-pod-123"}, vault.deleteSecretCalls)
	assert.Equal(t, []string{"my-pod-123"}, vault.removeSecretCalls)
	assert.Equal(t, []string{"policy-my-pod-123"}, vault.deletePolicyCalls)
}

func TestDeleteOrphan_ContinuesOnClientErrors(t *testing.T) {
	vault := &vaultSpy{
		deleteSecretErr: errors.New("delete failed"),
		removeSecretErr: errors.New("remove failed"),
		deletePolicyErr: errors.New("policy failed"),
	}
	client := NewClient(vault)

	err := client.DeleteOrphan(context.Background(), "my-pod")
	assert.NoError(t, err)
	assert.Len(t, vault.deleteSecretCalls, 1)
	assert.Len(t, vault.removeSecretCalls, 1)
	assert.Len(t, vault.deletePolicyCalls, 1)
}
