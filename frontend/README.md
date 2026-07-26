# Frontend (product web UI)

Production React application for Swyngora.

> **Status:** Project initialization scaffold is live on branch `feature/frontend-init`.  
> Track work in **`project-management/`**. Design: `docs/design/frontend-project-initialization.md`.  
> `simple-frontend/` remains a static API harness only.

## Stack (decided)

| Layer     | Library                                                   |
| --------- | --------------------------------------------------------- |
| UI kit    | **Ant Design** (`antd`)                                   |
| Charts    | **TradingView Lightweight Charts** (`lightweight-charts`) |
| Data      | RTK Query + OpenAPI → `src/libs/api`                      |
| Structure | Atomic Design UI + `src/libs/{api,hooks,utils}`           |

Decision: `project-management/decisions/001-antd-and-lightweight-charts.md`  
Design system: [`docs/design/frontend-design-system.md`](../docs/design/frontend-design-system.md)

### Brand palette

| Navy      | Indigo    | Steel     | Cream     |
| --------- | --------- | --------- | --------- |
| `#111844` | `#4B5694` | `#7288AE` | `#EAE0CF` |

Tokens: `src/styles/tokens/` · Theme: `src/styles/theme.ts` · Atoms: `Text`, `Skeleton`, `Button` (`isLoading` supported).

**Styling:** styled-components only. Colocate `ComponentName.styles.ts` — do not add `.css` / CSS modules.

## Conventions

| Rule                | Source                                  |
| ------------------- | --------------------------------------- |
| Package agent rules | `frontend/AGENTS.md`                    |
| System design       | `docs/design/frontend-system-design.md` |
| Local tasks / board | `project-management/board.md`           |
| First feature       | Multi-exchange spot markets             |

## Intended layout

```text
frontend/
├── README.md
├── AGENTS.md
├── package.json
└── src/
    ├── app/                 # ConfigProvider (antd), Redux, router
    ├── config/
    ├── components/          # Atomic UI (wrap antd / chart hosts)
    ├── features/markets/    # spot markets UI
    ├── libs/
    │   ├── api/             # RTK + generated OpenAPI
    │   ├── hooks/
    │   ├── utils/           # formatters, candle mappers
    │   └── types/
    └── styles/
```

## Run (local)

```bash
# terminal 1 — API
export PATH="$HOME/.local/go-sdk/go/bin:$PATH"   # if go not on PATH
cd backend && go run ./cmd/server                 # :8080

# terminal 2 — product UI
cd frontend && npm install && npm run dev         # :5174
```

Open http://localhost:5174 (from Linux) or **http://&lt;wsl-ip&gt;:5174** from Windows.

### WSL / Windows browser

If you open the UI from Windows as `http://172.x.x.x:5174`:

1. Keep **`VITE_API_BASE_URL` empty** (default) so the browser calls the **same origin**.
2. Vite proxies `/api` and `/health` to the Go API on `127.0.0.1:8080` **inside WSL**.
3. Do **not** use `http://localhost:8080` as the API base from a Windows browser — that hits Windows, not WSL.

Backend CORS already allows `*` for local dev.

Find WSL IP: `hostname -I | awk '{print $1}'`

## Format / lint

```bash
npm run format        # Prettier write
npm run format:check  # CI-friendly check
npm run lint          # ESLint (prettier-compatible)
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
