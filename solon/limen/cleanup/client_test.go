package cleanup

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

type elasticSpy struct {
	err       error
	usernames []string
}

func (e *elasticSpy) DeleteOrphan(username string) error {
	e.usernames = append(e.usernames, username)
	return e.err
}

type vaultSpy struct {
	err      error
	podNames []string
}

func (v *vaultSpy) DeleteOrphan(podName string) error {
	v.podNames = append(v.podNames, podName)
	return v.err
}

func TestDeleteOrphan_DeletesFromBothBackends(t *testing.T) {
	elastic := &elasticSpy{}
	vault := &vaultSpy{}
	client := NewClient(elastic, vault)

	err := client.DeleteOrphan("my123", "my-pod-123")
	assert.NoError(t, err)
	assert.Equal(t, []string{"my123"}, elastic.usernames)
	assert.Equal(t, []string{"my-pod-123"}, vault.podNames)
}

func TestDeleteOrphan_ReturnsCombinedErrors(t *testing.T) {
	elastic := &elasticSpy{err: errors.New("elastic failed")}
	vault := &vaultSpy{err: errors.New("vault failed")}
	client := NewClient(elastic, vault)

	err := client.DeleteOrphan("my123", "my-pod-123")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "elastic failed")
	assert.Contains(t, err.Error(), "vault failed")
}
