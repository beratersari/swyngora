# Frontend (product web UI)

Production React application for Swyngora.

> **Status:** Markets, coin detail, watchlist, **swing signals**, pumps, alerts, compare, and **AI chat** (`/ai`) on a trading-desk chrome (ticker tape, jump search, live status).  
> Track work in **`project-management/`**. Detail design: `docs/features/coin-detail.md`.  
> `simple-frontend/` remains a static API harness only.

## Stack (decided)

| Layer     | Library                                                   |
| --------- | --------------------------------------------------------- |
| UI kit    | **Ant Design** (`antd`)                                   |
| Charts    | **TradingView Lightweight Charts** (`lightweight-charts`) |
| Data      | RTK Query + OpenAPI → `src/libs/api`; live ticks via `src/libs/realtime` |
| Structure | Atomic Design UI + `src/libs/{api,realtime,hooks,utils}`  |

Decision: `project-management/decisions/001-antd-and-lightweight-charts.md`  
Design system: [`docs/design/frontend-design-system.md`](../docs/design/frontend-design-system.md)

### Brand palette

| Primary | HEX | Secondary | HEX | Neutral | HEX |
| --- | --- | --- | --- | --- | --- |
| Rich Black | `#000F0F` | Pine | `#063028` | Stone | `#707D7D` |
| Dark Green | `#032221` | Basil | `#0B453A` | Pistachio | `#AACBC4` |
| Bangladesh Green | `#03624C` | Forest | `#095544` | | |
| Mountain Meadow | `#4FD4A5` | Frog | `#17876D` | | |
| Caribbean Green | `#00FF81` | Mint | `#74F9BC` | | |
| Anti-Flash White | `#F1F7F6` | | | | |

Tokens: `src/styles/tokens/colors.ts` · Motion: `src/styles/tokens/motion.ts` · Theme: `src/styles/theme.ts` · Atoms: `Text`, `Skeleton`, `Button` (`isLoading` supported).

**Styling:** styled-components only. Colocate `ComponentName.styles.ts` — do not add `.css` / CSS modules.

## Conventions

| Rule                | Source                                  |
| ------------------- | --------------------------------------- |
| Package agent rules | `frontend/AGENTS.md`                    |
| System design       | `docs/design/frontend-system-design.md` |
| Local tasks / board | `project-management/board.md`           |
| Features            | Markets · detail · watchlist · pumps · AI chat |

## Intended layout

```text
frontend/
├── README.md
├── AGENTS.md
├── package.json
└── src/
    ├── app/                 # ConfigProvider (antd), Redux, router
    ├── config/
    ├── components/          # Atomic UI only (no features/)
    │   ├── atoms/
    │   ├── molecules/
    │   ├── organisms/       # domain sections (table, detail panels)
    │   ├── templates/
    │   └── pages/           # MarketsPage, CoinDetailPage (RTK here)
    ├── libs/
    │   ├── api/             # RTK + generated OpenAPI
    │   ├── hooks/
    │   ├── utils/           # formatters, candle mappers
    │   └── types/
    └── styles/
```

**Layout policy:** Option A — domain UI in `organisms/`, screens in `pages/`, shared logic in `libs/`. No `src/features/`.

### Localization

- **i18next** catalogs: `src/libs/i18n/locales/{en,tr}/`
- Namespaces: `common`, `markets`, `detail`, `watchlist`, `pumps`, `ai`, `alerts`, `compare`, `signals`
- Language switcher in the app header (persists to `localStorage`)
- Add a language: new locale folder + register in `libs/i18n/resources.ts` + `SUPPORTED_LOCALES`

## Run (local)

```bash
# terminal 1 — API
export PATH="$HOME/.local/go-sdk/go/bin:$PATH"   # if go not on PATH
cd backend && go run ./cmd/server                 # :8080

# terminal 2 — product UI
cd frontend && npm install && npm run dev         # :5174

# terminal 3 — AI (needed for /ai chat; optional for markets)
cd ai && uv sync && source .venv/bin/activate
# needs Ollama (local) or XAI_API_KEY + AI_LLM_PROVIDER=grok
export SWYNGORA_API_URL=http://127.0.0.1:8080
python -m swyngora_ai.serve --host 127.0.0.1 --port 8090
```

Open http://localhost:5174 (from Linux) or **http://&lt;wsl-ip&gt;:5174** from Windows.  
AI chat route: **http://localhost:5174/ai** (streams `POST /api/v1/ai/chat/stream` → Go → Python `:8090`, with `POST /api/v1/ai/chat` as fallback). A **Process** panel lists status / think / tool steps live, then collapses so the answer stays readable. Coin questions gather public web/X URLs into a **Sources** list under the reply.

### WSL / Windows browser

If you open the UI from Windows as `http://172.x.x.x:5174`:

1. Keep **`VITE_API_BASE_URL` empty** (default) so the browser calls the **same origin**.
2. Vite proxies `/api` and `/health` to the Go API on `127.0.0.1:8080` **inside WSL**.
3. Do **not** use `http://localhost:8080` as the API base from a Windows browser — that hits Windows, not WSL.

Backend CORS already allows `*` for local dev.

Find WSL IP: `hostname -I | awk '{print $1}'`

## Format / lint / test

```bash
npm run format         # Prettier write
npm run format:check   # CI-friendly check
npm run lint           # ESLint (prettier-compatible)
npm test               # Vitest unit/component tests
npm run test:coverage  # Coverage (line + branch) via @vitest/coverage-v8
npm run test:e2e       # Playwright smoke (see e2e/README.md; install browsers once)
```

Config: `.prettierrc.json` · ignore: `.prettierignore`

## Scripts

```bash
npm install
npm run dev
npm run build
npm test
npm run lint
npm run codegen:api   # → src/libs/api/generated/
```

## Environment

| Variable            | Default                 | Purpose    |
| ------------------- | ----------------------- | ---------- |
| `VITE_API_BASE_URL` | `http://localhost:8080` | API origin |

## Work order

1. **Epic A** — Project initialization (incl. antd + lightweight-charts install)
2. **Epic B** — Multi-exchange spot markets (Ant Table; no candles required)
3. Later — Coin detail with Lightweight Charts

Tasks: `project-management/tasks/`.
