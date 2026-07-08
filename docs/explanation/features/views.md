---
title: Views Feature
---

# Views feature

The `views` feature adds a **React SPA** powered by Vite. The frontend is embedded into the Go binary via `embed.FS` and served by the Echo server.

## What it provides

| File | Purpose |
|------|---------|
| `internal/adapters/http/web/views.go` | SPA serving + Vite proxy |
| `static/embed.go` | Go embed for built frontend assets |
| `views/package.json` | Frontend dependencies |
| `views/vite.config.js` | Vite dev server config |
| `views/src/` | React application source |

Development mode proxies to the Vite dev server for HMR. Production serves embedded static files with SPA fallback.
