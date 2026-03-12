# SENTINEL Frontend Agent — Quick Reference

## Owned Directories
/web/src/globe/, /web/src/map/, /web/src/list/, /web/src/alerts/, /web/src/areas/, /web/src/health/, /web/src/onboarding/, /web/src/components/, /web/src/hooks/, /web/src/types/, /web/index.html, /web/vite.config.ts, /web/tsconfig.json, /web/package.json

## OpenAPI-First Rule
Read /api/openapi.yaml BEFORE writing any data-fetching code. If you need an endpoint that doesn't exist, request the change — don't invent your own.

## CesiumJS Rules
- Enable entity clustering from the very first commit — not later
- Never render more than 5,000 entities on the globe at once
- Use Entity API (not Primitive API) for v1
- Use Cartesian3 not raw lat/lon for performance

## Three Badges (Non-Negotiable)
Every entity tooltip must show: source label, precision label, freshness timestamp.

## SSE Consumer
Build with proper useEffect cleanup. No memory leaks. Reconnection with Last-Event-ID.

## Hardware Budget
Browser tab RAM (CesiumJS): 800 MB max. JS bundle: 5 MB max. Visible entities: 5,000 max. CPU idle: 5%.

## Week 1 Task
1. Set up Vite + React + TypeScript + Tailwind project in /web/
2. Read /api/openapi.yaml (written by Backend Agent)
3. Generate TypeScript types from OpenAPI spec in /web/src/types/api.ts
4. Build SSE hook (useSSE.ts) with cleanup
5. Build CesiumJS globe component with clustering enabled
6. Build Leaflet 2D fallback map
7. Build event list view

## Key Rules
- File ownership is absolute — never write outside owned dirs
- Quiet by default — no sounds, no popups
- All styling uses Tailwind — no custom CSS unless Tailwind can't do it
- Precision enum: exact | polygon_area | approximate | text_inferred | unknown
- No ships in v1

## Operator
ed. Execute tasks without asking for confirmation.
