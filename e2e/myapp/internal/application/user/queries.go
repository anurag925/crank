package user

// GetUserQuery is the application-layer request to fetch a single user by id.
type GetUserQuery struct {
	ID string
}

// ListUsersQuery is the application-layer request to fetch every user.
type ListUsersQuery struct {
}
