package elastic

import (
	"context"
	"errors"
	"testing"

	"github.com/odysseia-greek/agora/aristoteles"
	elasticmodels "github.com/odysseia-greek/agora/aristoteles/models"
	"github.com/stretchr/testify/assert"
)

type accessSpy struct {
	deleteUserErr   error
	deletedUserName []string
}

func (a *accessSpy) CreateRole(ctx context.Context, name string, roleRequest elasticmodels.CreateRoleRequest) (bool, error) {
	return false, nil
}

func (a *accessSpy) CreateUser(ctx context.Context, name string, userCreation elasticmodels.CreateUserRequest) (bool, error) {
	return false, nil
}

func (a *accessSpy) ListUsers(ctx context.Context) ([]string, error) {
	return nil, nil
}

func (a *accessSpy) DeleteUser(ctx context.Context, name string) (bool, error) {
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

func TestDeleteOrphan_DeletesUser(t *testing.T) {
	access := &accessSpy{}
	client := NewClient(&elasticSpy{access: access})
	err := client.DeleteOrphan("my123")
	assert.NoError(t, err)
	assert.Equal(t, []string{"my123"}, access.deletedUserName)
}

func TestDeleteOrphan_ContinuesOnClientErrors(t *testing.T) {
	access := &accessSpy{deleteUserErr: errors.New("elastic down")}
	client := NewClient(&elasticSpy{access: access})
	err := client.DeleteOrphan("my123")
	assert.NoError(t, err)
	assert.Equal(t, []string{"my123"}, access.deletedUserName)
}
