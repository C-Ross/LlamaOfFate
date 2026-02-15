---
name: web-ui-development
description: Guide for developing the React web UI — components, styling, testing, and build tooling. Use this when asked to add components, modify the theme, write web tests, or work with the Vite/Tailwind/shadcn stack.
---

# Web UI Development

This skill covers the React web frontend in `web/`. The UI communicates with the Go backend via WebSocket and renders the game experience in a two-panel layout.

## Tech Stack

| Tool | Version | Purpose |
|------|---------|---------|
| Vite | 7.x | Build tool & dev server |
| React | 19.x | UI framework |
| TypeScript | 5.9.x | Type safety |
| Tailwind CSS | 4.x | Utility-first styling (v4 uses `@theme inline` + CSS variables) |
| shadcn/ui | new-york style | Component library (copies into `src/components/ui/`) |
| Vitest | 4.x | Test runner (shares Vite config) |
| React Testing Library | latest | Component testing |
| framer-motion | latest | Animations |

## Project Structure

```
web/
  index.html                  - Entry HTML
  package.json                - Dependencies & scripts
  vite.config.ts              - Vite config (plugins, proxy, aliases)
  vitest.config.ts            - Test config (merges vite.config.ts)
  tsconfig.json               - Root TS config (path aliases for editor)
  tsconfig.app.json           - App TS config (strict, bundler mode)
  components.json             - shadcn/ui configuration
  eslint.config.js            - ESLint flat config
  src/
    main.tsx                  - React entry point
    App.tsx                   - Root layout (two-panel)
    index.css                 - Tailwind + theme variables
    lib/utils.ts              - cn() helper (clsx + tailwind-merge)
    components/
      SidebarCard.tsx          - Reusable sidebar card wrapper
      ui/                      - shadcn/ui components (DO NOT edit manually)
    test/
      setup.ts                 - Vitest setup (jest-dom matchers)
      App.test.tsx             - App component tests
      SidebarCard.test.tsx     - SidebarCard tests
```

## Justfile Targets

```bash
just web-install    # npm install
just web-dev        # Start Vite dev server (port 5173)
just web-build      # Production build (tsc + vite build)
just web-test       # Run Vitest
just web-lint       # Run ESLint
just web-validate   # Lint + test + build
just validate       # Go + Web validation combined
```

## Path Aliases

The `@/` alias maps to `src/`. It is configured in three places (all required):

- `tsconfig.json` — editor resolution
- `tsconfig.app.json` — TypeScript compilation
- `vite.config.ts` — Vite bundling

```tsx
import { Button } from "@/components/ui/button"
import { SidebarCard } from "@/components/SidebarCard"
import { cn } from "@/lib/utils"
```

## Adding shadcn/ui Components

shadcn copies component source into `src/components/ui/`. Do not hand-edit these files.

```bash
cd web && npx shadcn@latest add <component-name>
```

Available components already installed: `button`, `card`, `scroll-area`, `input`, `badge`.

The `components.json` configures shadcn: style is `new-york`, no RSC, uses `@/components/ui` alias.

## Theming

### CSS Variables in `src/index.css`

The theme uses **neutral colors** with light/dark mode via `@media (prefers-color-scheme: dark)`. All colors use oklch.

**Standard shadcn variables:** `--background`, `--foreground`, `--card`, `--primary`, `--secondary`, `--muted`, `--accent`, `--destructive`, `--border`, `--input`, `--ring`, plus sidebar variants.

**Game-specific semantic colors** (registered in `@theme inline`):

| Variable | Purpose | Tailwind class |
|----------|---------|---------------|
| `--consequence-mild` | Mild consequences (amber) | `text-consequence-mild`, `bg-consequence-mild` |
| `--consequence-moderate` | Moderate consequences (orange) | `text-consequence-moderate` |
| `--consequence-severe` | Severe consequences (red) | `text-consequence-severe` |
| `--boost` | Boosts (green) | `text-boost` |
| `--fate-point` | Fate points (blue/indigo) | `text-fate-point` |

Each has tuned values for both light and dark mode.

### Fonts

| Variable | Font | Usage |
|----------|------|-------|
| `--font-heading` | Inter | Headings, buttons — `font-heading` |
| `--font-body` | Source Serif 4 | Body text, narrative — `font-body` |

Fonts are loaded via Google Fonts `@import` at the top of `index.css`.

### Adding Theme Colors

1. Add `--my-color` to both `:root` and `@media (prefers-color-scheme: dark)` blocks
2. Register in `@theme inline` as `--color-my-color: var(--my-color)`
3. Use as `text-my-color`, `bg-my-color`, etc.

## Layout

The app uses a two-panel flexbox layout:

- **Left panel** (flex-1): Chat area with header, scrollable message area, and input form
- **Right panel** (w-80, hidden below `lg`): Game sidebar with `SidebarCard` components

## Creating Components

### Custom Components

Place in `src/components/`. Follow these patterns:

```tsx
import { cn } from "@/lib/utils"
import type { ReactNode } from "react"

interface MyComponentProps {
  title: string
  children: ReactNode
  className?: string
}

export function MyComponent({ title, children, className }: MyComponentProps) {
  return (
    <div className={cn("base-classes", className)}>
      <h3 className="font-heading">{title}</h3>
      <div className="font-body">{children}</div>
    </div>
  )
}
```

Key conventions:
- Use `cn()` for conditional/merged class names
- Accept optional `className` prop for composability
- Use `font-heading` for headings/labels, `font-body` for content text
- Use semantic color classes (`text-primary`, `bg-card`, `text-muted-foreground`)
- Extract repeated card/panel patterns into reusable components (see `SidebarCard`)

### shadcn Components (ui/)

Do NOT edit files in `src/components/ui/` manually. To customize, either:
- Wrap them in a custom component
- Use `className` prop to override styles
- Use `cn()` to merge classes

## Writing Tests

Tests live in `src/test/` using Vitest + React Testing Library.

```tsx
import { render, screen } from "@testing-library/react"
import { describe, it, expect } from "vitest"
import { MyComponent } from "../components/MyComponent"

describe("MyComponent", () => {
  it("renders title and children", () => {
    render(<MyComponent title="Test">Content</MyComponent>)
    expect(screen.getByText("Test")).toBeInTheDocument()
    expect(screen.getByText("Content")).toBeInTheDocument()
  })
})
```

- `vitest.config.ts` merges from `vite.config.ts` (shares aliases, plugins)
- `setup.ts` imports `@testing-library/jest-dom/vitest` for DOM matchers
- `globals: true` — no need to import `describe`/`it`/`expect` (but explicit imports are fine)

## Vite Dev Server Proxy

`vite.config.ts` proxies `/ws` to `http://localhost:8080` (Go backend) with WebSocket upgrade support. During development, run both:

```bash
just serve      # Go backend on :8080
just web-dev    # Vite dev server on :5173 (proxies /ws → :8080)
```

## ESLint

Flat config in `eslint.config.js`. Includes:
- `@eslint/js` recommended
- `typescript-eslint` recommended
- `react-hooks` recommended
- `react-refresh` with `allowConstantExport: true` (needed for shadcn variant exports)

The 2 warnings on `badge.tsx`/`button.tsx` about constant exports are expected from shadcn-generated code.

## Common Pitfalls

- **Tailwind v4 uses `@theme inline`** — not `tailwind.config.js`. All theme extensions go in `index.css`.
- **Vitest config is separate** from `vite.config.ts` to avoid TS type conflicts with Vite 7's `defineConfig`. Use `vitest.config.ts` with `mergeConfig`.
- **Font `@import` must be first** in `index.css` (before `@import "tailwindcss"`) to avoid CSS ordering warnings.
- **shadcn `init` requires** both Tailwind's `@import "tailwindcss"` in CSS and `@/*` path alias in `tsconfig.json` before it will run.
- **`prefers-color-scheme`** drives dark mode — no `.dark` class toggle needed. Both light and dark variables are in `:root`.
