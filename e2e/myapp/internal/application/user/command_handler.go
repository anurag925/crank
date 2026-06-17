package user

import (
	"context"
	"fmt"

	"myapp/internal/domain/user"
	"myapp/internal/ports"
)

// CommandHandler is the application service that mutates user aggregates in
// response to Create/Update/Delete commands. It depends only on the domain
// Repository port and a UnitOfWork that wraps the save+publish atomicity
// boundary.
type CommandHandler struct {
	repo user.Repository
	uow  ports.UnitOfWork
}

// NewCommandHandler wires a CommandHandler against the given repository and
// unit of work.
func NewCommandHandler(repo user.Repository, uow ports.UnitOfWork) *CommandHandler {
	return &CommandHandler{repo: repo, uow: uow}
}

// HandleCreate creates a new user aggregate, persists it through the unit of
// work, and the UoW publishes the events it recorded during construction.
func (h *CommandHandler) HandleCreate(ctx context.Context, cmd CreateUserCommand) (*user.User, error) {
	id, err := user.NewUserID(cmd.ID)
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}
	u, err := user.NewUser(id, cmd.Name, cmd.Email)
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}
	if err := h.uow.SaveAndPublish(ctx, func(ctx context.Context) error { return h.repo.Save(ctx, u) }, u.PullEvents()); err != nil {
		return nil, fmt.Errorf("save user: %w", err)
	}
	return u, nil
}

// HandleUpdate loads an existing user aggregate, mutates it through the
// domain's Update method, and routes the save+publish through the unit of
// work so the write and recorded event are committed atomically.
func (h *CommandHandler) HandleUpdate(ctx context.Context, cmd UpdateUserCommand) (*user.User, error) {
	id, err := user.NewUserID(cmd.ID)
	if err != nil {
		return nil, fmt.Errorf("update user: %w", err)
	}
	u, err := h.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	u.Update(cmd.Name, cmd.Email)
	if err := h.uow.SaveAndPublish(ctx, func(ctx context.Context) error { return h.repo.Save(ctx, u) }, u.PullEvents()); err != nil {
		return nil, fmt.Errorf("save user: %w", err)
	}
	return u, nil
}

// HandleDelete removes a user aggregate. The delete is routed through the
// unit of work so the deletion and its recorded event are committed
// atomically.
func (h *CommandHandler) HandleDelete(ctx context.Context, cmd DeleteUserCommand) error {
	id, err := user.NewUserID(cmd.ID)
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	u, err := h.repo.Get(ctx, id)
	if err != nil {
		return err
	}
	u.MarkDeleted()
	if err := h.uow.SaveAndPublish(ctx, func(ctx context.Context) error { return h.repo.Delete(ctx, id) }, u.PullEvents()); err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	return nil
}
