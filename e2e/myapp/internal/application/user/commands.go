package user

// CreateUserCommand is the application-layer request to create a new user.
// The CommandHandler converts the stringly-typed ID into a typed UserID
// before constructing the aggregate.
type CreateUserCommand struct {
	ID    string
	Name  string
	Email string
}

// UpdateUserCommand is the application-layer request to mutate an existing
// user. The CommandHandler loads the aggregate by ID, calls Update, and
// persists the result.
type UpdateUserCommand struct {
	ID    string
	Name  string
	Email string
}

// DeleteUserCommand is the application-layer request to remove a user.
type DeleteUserCommand struct {
	ID string
}
