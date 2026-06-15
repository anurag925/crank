package web

import "github.com/labstack/echo/v4"

// MountConfig groups the per-resource HTTP handlers the central router
// mounts. Generated handlers (produced by `crank make handler` and
// `crank make scaffold`) are spliced into this struct at the field-splice
// marker.
type MountConfig struct {
	UserHandler *UserHandler
	// crank:http-fields (do not remove — `crank make handler` splices new fields here)
}

// Mount attaches every handler in cfg to e, each under its own group
// prefix. Generated handlers are spliced in at the register-splice marker.
func Mount(e *echo.Echo, cfg MountConfig) {
	g := e.Group("/users")
	cfg.UserHandler.Register(g)
	// crank:http-register (do not remove — `crank make handler` splices new route registrations here)
}
