# AFI docs site (Starlight)

Static documentation site built with [Astro Starlight](https://starlight.astro.build/).
Markdown content is synced from [`../docs`](../docs) — edit files there, then run sync/dev.

Mermaid fenced blocks (` ```mermaid `) are rendered via [astro-mermaid](https://github.com/joesaby/astro-mermaid), including light/dark theme switching.

## Local

```bash
cd docs-site
pnpm install
pnpm dev
```

Open http://localhost:4321

Or from the repo root:

```bash
make doc-starlight
```

## Build

```bash
pnpm build
```

Output is written to `dist/` (suitable for GitHub Pages).
