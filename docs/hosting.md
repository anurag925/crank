---
title: Hosting Documentation
---

# Hosting documentation with docmd

The `crank` documentation lives in the same repository as the code and is built with [docmd](https://docmd.io).

This keeps docs changes close to the code changes that require them.

## Files

```text
crank/
├── docs/
├── docmd.config.json
├── package.json
└── .github/workflows/docs.yml
```

| File | Purpose |
| --- | --- |
| `docs/` | Markdown source files. |
| `docmd.config.json` | Site title, description, source directory, output directory, and public URL. |
| `package.json` | Convenience scripts for local docs development. |
| `.github/workflows/docs.yml` | Builds the static site and deploys it to GitHub Pages. |

## Develop locally

```bash
npm run docs:dev
```

This runs:

```bash
npx @docmd/core dev
```

## Build locally

```bash
npm run docs:build
```

The output is written to:

```text
dist/
```

## Deploy on GitHub Pages

The GitHub Actions workflow runs on:

- pushes to `main` that touch docs files
- pull requests that touch docs files
- manual `workflow_dispatch`

Pull requests build the site to catch errors. Pushes to `main` also deploy to GitHub Pages.

## Enable GitHub Pages

In the GitHub repository settings:

1. Open **Settings**.
2. Open **Pages**.
3. Set **Build and deployment** source to **GitHub Actions**.
4. Push to `main` or run the docs workflow manually.

The configured public URL is:

```text
https://anurag925.github.io/crank
```

If you use a custom domain, update `docmd.config.json`.

## Writing guidelines

- Keep docs user-facing.
- Prefer examples that can be copied directly.
- Use exact feature and command names from the current code.
- Update docs in the same PR as CLI behavior changes.
- Avoid documenting speculative features as if they are complete.
