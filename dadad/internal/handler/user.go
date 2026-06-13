package handler

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"dadad/internal/model"
	"dadad/internal/service"
)

// UserHandler exposes HTTP endpoints for managing users. The actual data store is
// abstracted behind a service so handlers stay thin.
type UserHandler struct {
	svc *service.UserService
}

// NewUserHandler builds a UserHandler from the shared Deps.
func NewUserHandler(deps Deps) *UserHandler {
	return &UserHandler{svc: service.NewUserService()}
}

// Register attaches user routes to the Echo instance.
func (h *UserHandler) Register(e *echo.Echo) {
	g := e.Group("/users")
	g.GET("", h.List)
	g.POST("", h.Create)
	g.GET("/:id", h.Get)
}

// List godoc
//
//	@Summary      List users
//	@Description  Returns all users.
//	@Tags         users
//	@Produce      json
//	@Success      200  {array}   model.User
//	@Failure      500  {object}  map[string]string
//	@Router       /users [get]
func (h *UserHandler) List(c echo.Context) error {
	users := h.svc.List()
	return c.JSON(http.StatusOK, users)
}

// Create godoc
//
//	@Summary      Create a user
//	@Description  Creates a new user. The request body is validated automatically.
//	@Tags         users
//	@Accept       json
//	@Produce      json
//	@Param        user  body      model.User  true  "User payload"
//	@Success      201   {object}  model.User
//	@Failure      422   {object}  model.APIError
//	@Failure      500   {object}  map[string]string
//	@Router       /users [post]
func (h *UserHandler) Create(c echo.Context) error {
	var in model.User
	if err := c.Bind(&in); err != nil {
		return err
	}
	created := h.svc.Create(in)
	return c.JSON(http.StatusCreated, created)
}

// Get godoc
//
//	@Summary      Get a user
//	@Description  Returns a single user by ID.
//	@Tags         users
//	@Produce      json
//	@Param        id   path      int  true  "User ID"
//	@Success      200  {object}  model.User
//	@Failure      400  {object}  map[string]string
//	@Failure      404  {object}  map[string]string
//	@Router       /users/{id} [get]
func (h *UserHandler) Get(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid id"})
	}
	u, ok := h.svc.Get(id)
	if !ok {
		return c.JSON(http.StatusNotFound, echo.Map{"error": "user not found"})
	}
	return c.JSON(http.StatusOK, u)
}
