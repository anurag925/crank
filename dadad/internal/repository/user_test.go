package repository

import (
	"context"
	"testing"

	"dadad/internal/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserRepository_Create(t *testing.T) {
	repo := NewUserRepository()
	ctx := context.Background()

	u := &model.User{Name: "Alice", Email: "alice@example.com"}
	err := repo.Create(ctx, u)

	require.NoError(t, err)
	assert.Equal(t, int64(1), u.ID)
}

func TestUserRepository_CreateIncrementsID(t *testing.T) {
	repo := NewUserRepository()
	ctx := context.Background()

	u1 := &model.User{Name: "Alice", Email: "alice@example.com"}
	u2 := &model.User{Name: "Bob", Email: "bob@example.com"}

	require.NoError(t, repo.Create(ctx, u1))
	require.NoError(t, repo.Create(ctx, u2))

	assert.Equal(t, int64(1), u1.ID)
	assert.Equal(t, int64(2), u2.ID)
}

func TestUserRepository_List(t *testing.T) {
	repo := NewUserRepository()
	ctx := context.Background()

	users, err := repo.List(ctx)
	require.NoError(t, err)
	assert.Empty(t, users)

	_ = repo.Create(ctx, &model.User{Name: "Alice", Email: "alice@example.com"})
	_ = repo.Create(ctx, &model.User{Name: "Bob", Email: "bob@example.com"})

	users, err = repo.List(ctx)
	require.NoError(t, err)
	assert.Len(t, users, 2)
}

func TestUserRepository_Get(t *testing.T) {
	repo := NewUserRepository()
	ctx := context.Background()

	created := &model.User{Name: "Alice", Email: "alice@example.com"}
	_ = repo.Create(ctx, created)

	user, err := repo.Get(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, "Alice", user.Name)
	assert.Equal(t, "alice@example.com", user.Email)
}

func TestUserRepository_GetNotFound(t *testing.T) {
	repo := NewUserRepository()
	ctx := context.Background()

	_, err := repo.Get(ctx, 999)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrUserNotFound)
}
