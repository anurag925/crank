package user

import (
	"context"
	"fmt"

	"myapp/internal/domain/user"
)

// QueryHandler is the application service that reads user aggregates in
// response to Get/List queries. It depends only on the domain Repository port.
type QueryHandler struct {
	repo user.Repository
}

// NewQueryHandler wires a QueryHandler against the given repository.
func NewQueryHandler(repo user.Repository) *QueryHandler {
	return &QueryHandler{repo: repo}
}

// HandleGet fetches a single user aggregate by id.
func (h *QueryHandler) HandleGet(ctx context.Context, q GetUserQuery) (*user.User, error) {
	id, err := user.NewUserID(q.ID)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	return h.repo.Get(ctx, id)
}

// HandleList returns every persisted user aggregate.
func (h *QueryHandler) HandleList(ctx context.Context, _ ListUsersQuery) ([]*user.User, error) {
	return h.repo.List(ctx)
}
