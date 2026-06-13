package service

import (
	"testing"

	"dadad/internal/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserService_Create(t *testing.T) {
	svc := NewUserService()

	user := svc.Create(model.User{Name: "Alice", Email: "alice@example.com"})

	assert.Equal(t, int64(1), user.ID)
	assert.Equal(t, "Alice", user.Name)
	assert.Equal(t, "alice@example.com", user.Email)
}

func TestUserService_CreateIncrementsID(t *testing.T) {
	svc := NewUserService()

	u1 := svc.Create(model.User{Name: "Alice", Email: "alice@example.com"})
	u2 := svc.Create(model.User{Name: "Bob", Email: "bob@example.com"})

	assert.Equal(t, int64(1), u1.ID)
	assert.Equal(t, int64(2), u2.ID)
}

func TestUserService_List(t *testing.T) {
	svc := NewUserService()

	users := svc.List()
	assert.Empty(t, users)

	svc.Create(model.User{Name: "Alice", Email: "alice@example.com"})
	svc.Create(model.User{Name: "Bob", Email: "bob@example.com"})

	users = svc.List()
	assert.Len(t, users, 2)
}

func TestUserService_Get(t *testing.T) {
	svc := NewUserService()

	created := svc.Create(model.User{Name: "Alice", Email: "alice@example.com"})

	user, ok := svc.Get(created.ID)
	require.True(t, ok)
	assert.Equal(t, "Alice", user.Name)
}

func TestUserService_GetNotFound(t *testing.T) {
	svc := NewUserService()

	_, ok := svc.Get(999)
	assert.False(t, ok)
}
