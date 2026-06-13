package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"bytes"
	"testing"

	"dadad/internal/model"
	"dadad/internal/validator"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestEcho() *echo.Echo {
	e := echo.New()
	e.HideBanner = true
	e.Binder = &echoBinder{defaultBinder: new(echo.DefaultBinder)}
	return e
}

func TestUserHandler_List_Empty(t *testing.T) {
	e := setupTestEcho()
	req := httptest.NewRequest(http.MethodGet, "/users", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	h := NewUserHandler(Deps{})
	err := h.List(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var users []model.User
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &users))
	assert.Empty(t, users)
}
// When auth is enabled, User.Password has json:"-" so the JSON binder
// cannot populate it — the custom validator then rejects the empty Password.
// User creation should go through /auth/register in that case. These Create
// and Get tests only run without the auth feature.

func TestUserHandler_Create(t *testing.T) {
	e := setupTestEcho()

	body, _ := json.Marshal(model.User{Name: "Alice", Email: "alice@example.com"})
	req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	h := NewUserHandler(Deps{})
	err := h.Create(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, rec.Code)

	var user model.User
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &user))
	assert.Equal(t, "Alice", user.Name)
	assert.Equal(t, int64(1), user.ID)
}

func TestUserHandler_Create_ValidationError(t *testing.T) {
	e := setupTestEcho()

	// Missing required name and invalid email.
	body, _ := json.Marshal(map[string]string{"email": "bad"})
	req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	h := NewUserHandler(Deps{})
	err := h.Create(c)

	// The custom binder should return a ValidationError.
	require.Error(t, err)
	_, ok := err.(*validator.ValidationError)
	assert.True(t, ok, "expected *validator.ValidationError")
}

func TestUserHandler_Get(t *testing.T) {
	e := setupTestEcho()

	// Create a user first.
	createBody, _ := json.Marshal(model.User{Name: "Alice", Email: "alice@example.com"})
	createReq := httptest.NewRequest(http.MethodPost, "/users", bytes.NewReader(createBody))
	createReq.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	createRec := httptest.NewRecorder()
	createCtx := e.NewContext(createReq, createRec)

	h := NewUserHandler(Deps{})
	_ = h.Create(createCtx)

	var created model.User
	_ = json.Unmarshal(createRec.Body.Bytes(), &created)

	// Now get the user.
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/users/:id")
	c.SetParamNames("id")
	c.SetParamValues("1")

	err := h.Get(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var user model.User
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &user))
	assert.Equal(t, "Alice", user.Name)
}

func TestUserHandler_Get_NotFound(t *testing.T) {
	e := setupTestEcho()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/users/:id")
	c.SetParamNames("id")
	c.SetParamValues("999")

	h := NewUserHandler(Deps{})
	err := h.Get(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestUserHandler_Get_InvalidID(t *testing.T) {
	e := setupTestEcho()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/users/:id")
	c.SetParamNames("id")
	c.SetParamValues("abc")

	h := NewUserHandler(Deps{})
	err := h.Get(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUserHandler_RegisterRoutes(t *testing.T) {
	e := setupTestEcho()
	h := NewUserHandler(Deps{})
	h.Register(e)

	// Inspect the registered routes.
	routes := e.Routes()
	routeMap := make(map[string]bool)
	for _, r := range routes {
		routeMap[r.Method+" "+r.Path] = true
	}

	assert.True(t, routeMap["GET /users"], "GET /users route should exist")
	assert.True(t, routeMap["GET /users/:id"], "GET /users/:id route should exist")
	assert.True(t, routeMap["POST /users"], "POST /users route should exist")
}
