---
title: Views Feature
---

# Views feature

The `views` feature adds a **React SPA** powered by Vite. The frontend is embedded into the Go binary via `embed.FS` and served by the Echo server, with dev-mode hot-reload proxied to the Vite dev server.

## What it provides

| File | Purpose |
|------|---------|
| `internal/adapters/http/web/views.go` | SPA serving + Vite proxy middleware |
| `static/embed.go` | Go embed for built frontend assets |
| `views/package.json` | Frontend dependencies |
| `views/vite.config.js` | Vite dev server configuration |
| `views/index.html` | HTML entry point |
| `views/src/main.jsx` | React entry point |
| `views/src/App.jsx` | React application component |
| `views/src/App.css` | Application styles |
| `views/src/api.js` | API client module |

### How serving works

- **Development**: Echo proxies `/` requests to the Vite dev server (HMR enabled).
- **Production**: Vite builds to `views/dist/`, which is embedded via `static/embed.go` and served directly by Echo.
- Toggle between modes with `views.enabled` and `views.dev_server` in config.

## Tech stack

| Library | Purpose | Documentation |
|---------|---------|---------------|
| [React](https://react.dev) | UI component library | [react.dev](https://react.dev) |
| [Vite](https://vitejs.dev) | Frontend build tool and dev server | [vitejs.dev](https://vitejs.dev) |
| [Go embed](https://pkg.go.dev/embed) | Embedded static file serving | [pkg.go.dev/embed](https://pkg.go.dev/embed) |

## Learning resources

- **React documentation** — [react.dev](https://react.dev) (learn, reference, tutorials)
- **Vite getting started** — [vitejs.dev/guide](https://vitejs.dev/guide/)
- **Go embed documentation** — [pkg.go.dev/embed](https://pkg.go.dev/embed)
- **React + Vite tutorial** — [vitejs.dev/guide/#scaffolding-your-first-vite-project](https://vitejs.dev/guide/#scaffolding-your-first-vite-project)
- **shadcn/ui** — [ui.shadcn.com](https://ui.shadcn.com) (popular component library for React + Vite)

## Notes

- The frontend source lives under `views/` at the project root, not inside `internal/`.
- In development, Vite HMR provides instant hot module replacement — edit React components and see updates in real time.
- The Go embed directive reads from `static/dist/`, which is where the built frontend assets are copied during deployment.
- This feature adds zero Go dependencies — the `static/embed.go` uses only the standard library.
