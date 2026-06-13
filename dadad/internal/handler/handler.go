package handler

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"dadad/internal/validator"
)

// Deps groups optional dependencies a handler can rely on. Fields are populated by
// the feature that owns them (e.g. postgres fills DB, auth fills JWT).
type Deps struct {
}

// Handler carries shared dependencies and exposes route registration helpers.
type Handler struct {
	deps Deps
	users *UserHandler
	// crank:handler-fields (do not remove — `crank make handler` inserts fields here)
}

// New constructs a Handler. Feature-specific dependencies are wired in via Deps.
func New(deps Deps) *Handler {
	return &Handler{
		deps:  deps,
		users: NewUserHandler(deps),
		// crank:handler-init (do not remove — `crank make handler` inserts initializers here)
	}
}

// Register attaches all routes owned by the handler set to the Echo instance.
func (h *Handler) Register(e *echo.Echo) {
	// Replace the default binder with one that runs struct validation
	// automatically after binding JSON bodies.
	e.Binder = &echoBinder{defaultBinder: new(echo.DefaultBinder)}

	e.GET("/health", h.Health)
	h.users.Register(e)
	// crank:handler-register (do not remove — `crank make handler` inserts route registrations here)
}

// Health godoc
//
//	@Summary      Liveness probe
//	@Description  Returns server status and current UTC time. Suitable for Kubernetes or load balancer health checks.
//	@Tags         system
//	@Produce      json
//	@Success      200  {object}  map[string]string
//	@Router       /health [get]
func (h *Handler) Health(c echo.Context) error {
	return c.JSON(http.StatusOK, echo.Map{
		"status": "ok",
		"time":   time.Now().UTC().Format(time.RFC3339),
	})
}

// echoBinder wraps the default Echo binder so that struct validation is applied
// automatically on every Bind() call. Handlers no longer need to call the
// validator manually — if Bind() returns nil the input is guaranteed valid.
type echoBinder struct {
	defaultBinder echo.Binder
}

func (b *echoBinder) Bind(i any, c echo.Context) error {
	if err := b.defaultBinder.Bind(i, c); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if err := validator.Struct(i); err != nil {
		return err
	}
	return nil
}
