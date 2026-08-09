# AGENTS.md

Instructions for humans and coding agents working on **Swyngora**.

Treat this file as the source of truth for collaboration, branching, versioning, and day-to-day engineering defaults. Prefer updating this document when conventions change rather than relying on tribal knowledge.

---

## 1. Project overview

**Swyngora** is an AI-powered cryptocurrency (and stock) analysis platform. Market data moves quickly; most products only expose basic quotes. Swyngora aims to close that gap with:

| Capability | Description |
|---|---|
| Market data | Real-time crypto and stock data |
| Cross-exchange insights | Compare trading volume for a coin across exchanges |
| Exchange views | List/sort coins on a single exchange by market capitalization |
| Analytics | Technical indicators, swing trading signals, risk analysis |
| Trading tools | Paper trading, watchlists, alerts |
| AI assistant | Answers questions, explains market data, remembers prior conversation context for better suggestions |

### Clients

| Surface | Stack | Notes |
|---|---|---|
| Simple frontend | Static HTML/JS | **Test harness only** under `simple-frontend/` — not Atomic Design / RTK Query |
| Web app (product) | React | Lives under `frontend/`; Atomic Design (§6.8); **Ant Design**; **Lightweight Charts**; **RTK Query**; OpenAPI types (§6.9); libs under `src/libs/` |
| Mobile app | React Native | `mobile/`; Atomic Design + **modules/pages + ViewModel**; **no Expo**; **Chrome via react-native-web** (`npm run web`); **RTK Query** + OpenAPI (§6.8 / §6.9); brand tokens match frontend |
| Messaging | Telegram bot | Optional transport under `backend/internal/transport/telegram` (same process as HTTP API; no AI) |

### Backend and AI

| Layer | Stack | Notes |
|---|---|---|
| API / services | **Go** | N-layered (§6.7); public HTTP API described by **OpenAPI** for client codegen (§6.9) |
| AI assistant | **Python** + **LangGraph/LangChain** | Multi-agent orchestrator under `ai/` (market, web, X, analyst specialists) |
| LLM providers | **Local Ollama** and **Grok (xAI)** only | See §6.5 — do not integrate other commercial LLM APIs as defaults |
| Tooling for AI | Custom **MCP** (Go server `backend/cmd/mcp`) | Market/watchlist tools for agents; mirror HTTP OpenAPI (§6.5) |

**Product goals:** fast, smart, easy-to-use analysis with modern AI.

---

## 2. Intended repository layout

The monorepo will grow over time. Target top-level layout (create packages only when needed; do not invent empty scaffolding without a task):

```text
swyngora/
├── AGENTS.md                 # This file — agent & team conventions
├── README.md                 # Human-facing project overview
├── VERSION                   # Current release version (SemVer), e.g. 0.1.0
├── CHANGELOG.md              # Keep a Changelog format
├── docs/                     # Architecture, ADRs, API notes
├── backend/                  # Go services and APIs (N-layered — see §6.7)
├── ai/                       # Python LangChain assistant + MCP tools
├── simple-frontend/          # Static test harness for the API (not production UI)
├── frontend/                 # Production web UI (reserved; Atomic Design when scaffolded)
├── web/                      # Optional alias/legacy name — prefer frontend/ for product UI
├── mobile/                   # React Native (Atomic Design — §6.8)
├── bot/                      # (unused) Telegram lives in backend/internal/transport/telegram
├── packages/                 # Shared libs (schemas, clients, types) if needed
├── project-management/       # Local epics/tasks/board (frontend work tracking)
└── scripts/                  # Dev, release, and CI helpers
```

**Frontend naming:** use `simple-frontend/` for lightweight API testing. The real product web app lives under `frontend/` (Atomic Design + RTK Query + OpenAPI types — §6.8 / §6.9). Do not treat `simple-frontend` as the long-term design-system home.

When a package gains its own long-lived conventions, add a nested `AGENTS.md` in that package. **Closest `AGENTS.md` wins** for files under that tree; user chat instructions always override docs.

---

## 3. Git Flow

We follow **Git Flow** adapted for a default branch named `main` (not `master`).

### 3.1 Long-lived branches

| Branch | Purpose | Protection |
|---|---|---|
| `main` | Production-ready code only. Every commit on `main` should be releasable. | Protected; no direct commits. Merge only via completed release or hotfix. |
| `develop` | Integration branch for the next release. Default target for feature work. | Protected; merge via MR/PR after review. |

### 3.2 Supporting branches

| Type | Naming | Branched from | Merges into | When to use |
|---|---|---|---|---|
| Feature | `feature/<short-description>` | `develop` | `develop` | New functionality, refactors, non-urgent work |
| Release | `release/vX.Y.Z` | `develop` | `main` **and** `develop` | Stabilize version, bump version, changelog, release-only fixes |
| Hotfix | `hotfix/vX.Y.Z` | `main` | `main` **and** `develop` | Critical production fixes |
| Bugfix | `bugfix/<short-description>` | `develop` | `develop` | Non-critical bugs (optional alias of feature flow) |
| Chore / docs | `chore/<short-description>` or `docs/<short-description>` | `develop` | `develop` | Tooling, CI, docs-only |

**Naming rules**

- Use lowercase kebab-case: `feature/paper-trading-orders`, `hotfix/v1.2.1`.
- Keep names short and intent-revealing (what, not who).
- One primary concern per branch; avoid mega-branches that mix unrelated work.
- Do **not** commit secrets, API keys, or real user data to any branch.

### 3.3 Day-to-day workflow (features)

```text
1. git checkout develop && git pull origin develop
2. git checkout -b feature/<short-description>
3. Implement + test + commit (see §5)
4. Push branch and open MR/PR → target: develop
5. Review, CI green, merge (prefer squash or merge commit — see §3.6)
6. Delete remote feature branch after merge
```

### 3.4 Release workflow

```text
1. git checkout develop && git pull
2. git checkout -b release/vX.Y.Z
3. Bump VERSION, update CHANGELOG.md, fix release-only issues (no new features)
4. Open MR/PR: release/vX.Y.Z → main
5. After merge to main: tag vX.Y.Z on main
6. Merge main (or the release branch) back into develop so version bumps land there too
7. Deploy from the tag / main as appropriate
```

### 3.5 Hotfix workflow

```text
1. git checkout main && git pull
2. git checkout -b hotfix/vX.Y.Z   # patch bump only
3. Fix + test + update CHANGELOG for the patch
4. MR/PR → main, merge, tag vX.Y.Z
5. Merge hotfix (or main) back into develop immediately
```

### 3.6 Merge policy

- **Never** force-push `main` or `develop`.
- Prefer **merge commits** or **squash** consistently; pick one team default and stick to it. Recommended default:
  - Features → **squash merge** into `develop` (clean history).
  - Releases / hotfixes → **merge commit** into `main` (preserve release points).
- **Squash / merge commit messages** must use the same Conventional Commits form as §5.1 (not `[scope] Title` or free-form MR UI defaults). Set the squash commit subject explicitly when merging.
- Rebase feature branches onto latest `develop` before merge when history is linear and shared history allows it; do not rewrite others’ published history without agreement.
- Keep MRs/PRs reviewable: small, focused, with a clear description and test plan.

### 3.7 Branch protection (recommended)

- Require MR/PR + at least one approval for `main` and `develop`.
- Require CI to pass before merge.
- Disallow direct pushes to `main` and `develop`.
- Restrict who may create tags matching `v*`.

### 3.8 Remotes (GitLab primary + GitHub mirror)

Swyngora is mirrored to **two** remotes. All humans and agents must keep them in sync when publishing work.

| Remote name | Host | Role |
|---|---|---|
| **`origin`** | GitLab (`nova.teachx.ai` / `trace-analysis/swyngora`) | **Primary** — team MRs, CI, protected `main` / `develop`, default `git pull` / tracking |
| **`beratersari`** | GitHub (`https://github.com/beratersari/swyngora`) | **Private mirror** — same history for backup / personal access; remote name matches the GitHub account |

**Rules**

1. **Always push published branches to both remotes** after a successful commit set that is meant to be shared (feature branches, `develop` after merge, `main` after release/hotfix, tags).
2. Prefer the alias (configure once per clone):

   ```bash
   git config alias.pushboth '!f(){ git push origin "$@"; git push beratersari "$@"; }; f'
   # examples:
   git pushboth -u HEAD
   git pushboth develop
   git pushboth main
   git pushboth --tags
   ```

   Without the alias: `git push origin <ref>` then `git push beratersari <ref>`.
3. **Open merge requests on GitLab (`origin`)** as the team workflow. GitHub is a mirror unless the team explicitly says otherwise — do not assume GitHub PRs replace GitLab MRs.
4. **Branch tracking** stays on `origin` (e.g. `develop` tracks `origin/develop`). Do not retarget long-lived branches to the GitHub remote for daily pull/rebase.
5. **First-time clone setup:**

   ```bash
   git clone <gitlab-ssh-or-https-url> swyngora
   cd swyngora
   git remote add beratersari https://github.com/beratersari/swyngora.git
   git config alias.pushboth '!f(){ git push origin "$@"; git push beratersari "$@"; }; f'
   git fetch beratersari
   ```

6. **New clone / new machine:** if `beratersari` is missing, add it before the next push. Agents must not silently drop the GitHub mirror.
7. **Long-lived branches on both remotes:** keep at least `main` and `develop` present on GitLab and GitHub. After the first release flow lands commits on `main`, push `main` (and tags) to both.
8. **Never** put secrets in either remote. Private on GitHub does not mean safe for keys or `.env` files.

**Why `main` may look empty early:** integration work lands on `develop` first (Git Flow). `main` only moves on release/hotfix. Still mirror whatever exists on `origin/main` to `beratersari` so both remotes have the full branch set.

---

## 4. Versioning

We use **[Semantic Versioning 2.0.0](https://semver.org/)** (`MAJOR.MINOR.PATCH`).

### 4.1 Meaning of each segment

| Change type | Bump | Examples |
|---|---|---|
| Breaking API / protocol / data contract change | **MAJOR** | Removing a public REST field, incompatible MCP tool schema, auth flow break |
| New backward-compatible feature | **MINOR** | New indicator, watchlist endpoint, AI tool |
| Backward-compatible bug fix | **PATCH** | Incorrect volume calculation, crash fix |
| Pre-1.0 development | `0.y.z` | Anything may change; still document breaks in changelog |

Until the first public stable release, start at **`0.1.0`** and advance MINOR for meaningful milestones, PATCH for fixes.

### 4.2 Where the version lives

| Artifact | Rule |
|---|---|
| Root `VERSION` | Single source of truth for the product release, plain text `X.Y.Z` |
| Git tags | Annotated tags: `vX.Y.Z` (e.g. `v0.1.0`) created on `main` only |
| `CHANGELOG.md` | [Keep a Changelog](https://keepachangelog.com/) sections: Added, Changed, Deprecated, Removed, Fixed, Security |
| Language manifests | Align package versions with the product tag when publishing clients/libs (e.g. Go module tags, npm package versions) |

### 4.3 Pre-release and build metadata (optional)

- Pre-releases: `v1.2.0-rc.1`, `v1.2.0-beta.1` (from `release/*` when needed).
- Do not use pre-release tags for production deploys unless explicitly approved.

### 4.4 Changelog discipline

- Every user-visible change that ships in a release must have a changelog entry.
- Write for operators and teammates: what changed and why it matters, not a dump of commit hashes.
- Security fixes get a clear **Security** subsection; never hide breaking changes.

### 4.5 Release checklist (agents and humans)

1. Confirm target SemVer bump from changes since last tag.
2. Update `VERSION` and `CHANGELOG.md` on the `release/*` or `hotfix/*` branch.
3. Ensure tests/lint pass for affected packages.
4. Merge to `main`, create annotated tag `vX.Y.Z`.
5. Merge back to `develop`.
6. Note deploy steps in the MR/PR if not fully automated.

---

## 5. Commits and merge requests

### 5.1 Conventional Commits (commits **and** MR/PR titles)

**One convention for both.** Git commits, MR/PR titles, and the message written onto `develop` / `main` on squash or merge must all use [Conventional Commits](https://www.conventionalcommits.org/). Do **not** use a separate `[scope] Title` form for MRs.

```text
<type>(<optional-scope>): <short imperative summary>

[optional body]

[optional footer]
```

**Types:** `feat`, `fix`, `docs`, `style`, `refactor`, `perf`, `test`, `build`, `ci`, `chore`, `revert`

**Scopes (examples):** `backend`, `ai`, `frontend`, `web`, `mobile`, `bot`, `mcp`, `docs`, `release`

**Examples (valid for commits and MR titles)**

```text
feat(backend): add cross-exchange volume comparison endpoint
fix(frontend): harden chart history, pump markers, and build
fix(ai): prevent assistant from leaking API keys in tool output
docs: document Git Flow and SemVer in AGENTS.md
chore(release): bump version to 0.2.0
```

**Rules**

- Subject line: imperative mood, ~72 chars, no trailing period.
- Breaking changes: add `!` after type/scope (`feat(api)!: ...`) and/or a `BREAKING CHANGE:` footer.
- Prefer a scope when the change is package-local (`frontend`, `backend`, `mobile`, `ai`, …).
- **Agents:** when opening an MR, set the title to the same Conventional Commits subject as the primary commit (or the intended squash message)—never `[backend] Add …` or other non-conventional titles.
- **Squash merge:** the squash commit subject on `develop` must be Conventional Commits (GitLab/GitHub default to the MR title—so the MR title must already be correct).

### 5.2 Merge request / pull request template (minimum)

Every MR/PR should include:

1. **Title** — Conventional Commits subject only (§5.1), e.g. `fix(frontend): harden chart history and pump markers`.
2. **Summary** — what and why (1–3 sentences).
3. **Type** — feature / fix / release / hotfix / chore / docs (should match the title `type`).
4. **Tests** — list new/updated test files and commands run (see §6.3). Features and behavioral fixes without tests are incomplete.
5. **Documentation** — list docs added/updated for new folders or features (see §8.1), and any **`AGENTS.md` / `README.md`** updates for stale guidance (see §8.2). Note N/A only when nothing in those files was affected.
6. **Risk / rollback** — especially for market-data and trading-adjacent code.
7. **Changelog note** — draft bullet if user-visible.
8. **Screenshots** — for UI changes (web/mobile).

---

## 6. Engineering principles for agents

### 6.1 Before coding

1. Read this file and the nearest package `AGENTS.md` / README.
2. Prefer the smallest change that solves the stated problem.
3. Do not expand scope (new frameworks, drive-by refactors, unsolicited docs) unless asked.
4. Match existing style in the package you touch.
5. Ask before destructive git operations, force-pushes, or changing shared CI/infra.
6. Plan **tests** and **docs** with the feature—not as an afterthought after the MR is open.
7. After any code change, update related **`AGENTS.md`** and **`README.md`** files (see §8.2)—same change set, not a follow-up.
8. **Restart the running backend yourself** after changes that affect `cmd/server` / transport / services (kill port `:8080` process and start `go run ./cmd/server`). Do not only print restart instructions for the user when you have shell access. Same for MCP: it is served by that single process at `/mcp`.

### 6.2 Implementation defaults

| Area | Default |
|---|---|
| Go backend | Idiomatic Go; **always N-layered architecture** (§6.7); **OpenAPI** as source of truth for public HTTP contracts (§6.9); context-aware handlers; structured logging; no panics in request paths |
| Python AI | Typed where practical; LangChain patterns consistent with existing agents; isolate prompts and tool schemas; **Ollama + Grok only** (§6.5) |
| MCP tools | Explicit schemas, least-privilege access, timeouts, no secret leakage in tool responses |
| React / RN | Atomic Design + file split (§6.8); product web uses **Ant Design** + **Lightweight Charts**; **RTK Query** for API data; **OpenAPI-generated** types (§6.9); non-UI code in **`frontend/src/libs/`**; background refresh + cache hygiene (§6.6) |
| APIs | Versioned public contracts via **OpenAPI**; validate input; never trust client-provided market “truth” without server-side checks |
| External data APIs | Prefer free/public endpoints; **do not build around paid pricing tiers or paid plan “choices”** (§6.5) |
| Caching | TTL, invalidation, and cleanup—especially for background runs (§6.6) |
| Config | Environment variables / secret managers; never hardcode keys or exchange credentials |

### 6.3 Testing — required for every feature and production file

**Rule: no feature ships without tests. New or materially changed production code must land with corresponding tests in the same MR/PR.**

| Situation | Required |
|---|---|
| New feature | Automated tests covering happy path + important edge cases / error paths |
| New production source file (logic, handlers, tools, services, reducers, etc.) | Companion tests for that unit’s public behavior |
| Bug fix | Regression test that fails before the fix and passes after |
| Pure docs, comments, formatting, or config with no behavior change | Tests N/A (state why in the MR) |
| Generated code / thin re-exports | Prefer testing the consumer or generator; do not add empty test shells |

**Placement and naming (follow package conventions once present)**

| Stack | Convention |
|---|---|
| Go (`backend/`) | `foo.go` → `foo_test.go` (same package or `_test` package as appropriate); test each layer—domain/use case with fakes, handlers with mocked services (§6.7) |
| Python (`ai/`) | `module.py` → `tests/test_module.py` or colocated `test_*.py` matching project layout |
| React / RN | Component or module → `*.test.ts(x)` / `*.spec.ts(x)` next to source or under `__tests__/`; colocate with Atomic Design folders (§6.8) |
| MCP tools | Schema validation + tool behavior tests (mocked backend/market clients) |
| Telegram bot | Handler/command unit tests; mock Telegram API |

**What “good enough” means**

- Cover the **public behavior** of the feature (API contract, pure functions, tool I/O), not every private line.
- Prefer unit tests for pure logic (indicators, ranking, risk math, parsers) and integration/API tests for HTTP or service boundaries.
- Use fakes/mocks for external exchanges, brokers, and LLMs—never hit live external network APIs in unit tests (mock Ollama/Grok clients too).
- Do not claim “all tests pass” without running them.
- UI: automated tests when the suite exists; otherwise document manual steps **and** still test non-UI logic (hooks, formatters, state machines) automatically.
- If a package has no test runner yet, **introduce the minimal harness in the same MR** as the first tested feature—do not accumulate untested code “until later.”

**Definition of done (features)**

1. Implementation complete.
2. Tests added/updated and green locally (and CI when available).
3. Documentation updated per §8.1 when a new folder or user-facing/capability surface was introduced.
4. Related **`AGENTS.md`** and **`README.md`** files updated for anything the change made stale (§8.2).
5. MR template sections for tests and docs filled in.

### 6.4 Security and compliance (non-negotiable)

- **No real money trading** in early versions unless explicitly designed; paper trading must be clearly labeled.
- Never commit `.env`, private keys, exchange API secrets, JWT signing keys, or production dumps.
- Treat AI tool outputs as untrusted for authorization decisions; enforce auth in the backend.
- Rate-limit external market data calls; cache with TTL and invalidation (§6.6); respect exchange ToS and API limits.
- Do not log PII, full auth tokens, or raw secrets.
- If generating financial insights, avoid absolute guarantees; analysis is informational, not financial advice—surface that in user-facing copy when adding product text.

### 6.5 AI providers, external APIs, and MCP

#### LLM providers (mandatory)

**Use only local Ollama and Grok (xAI) for model inference.** Do not add OpenAI, Anthropic, Gemini, or other commercial LLM SDKs/endpoints as product defaults, “fallbacks,” or pricing-tier options unless the team explicitly revises this document.

| Provider | Role | Notes |
|---|---|---|
| **Ollama** (local) | Default for local/dev and self-hosted inference | Configure base URL and model name via env; no cloud billing path |
| **Grok** (xAI) | Cloud LLM when remote models are needed | Use official xAI/Grok APIs; keys via env/secret manager only |

- LangChain (or equivalent) integrations must target **Ollama** and **Grok** providers only.
- Abstract the LLM behind a small internal interface so switching between Ollama and Grok is config-driven—not a user-facing “pick a paid plan” feature.
- Do **not** implement model/provider selection UX or backend enums based on **pricing tiers**, subscription SKUs, or paid API product lines.
- Do not document or hardcode paid third-party LLM pricing in code, config samples, or product copy.

#### External market / data APIs

When fetching market or other data from third-party APIs:

- **Do not use pricing-based choices** — no paid plan selection, premium tier flags, or code paths that assume a billable data-vendor package unless the team later approves a specific provider in an ADR.
- Prefer **free, public, or self-hosted** data sources and endpoints that do not require commercial pricing decisions in the product.
- Do not add UI or config for “Basic / Pro / Enterprise API plan” style data providers.
- Respect rate limits and ToS; cache with TTL and refresh/cleanup in background (§6.6); keep API keys out of the repo.
- If a free tier is later insufficient, document the constraint and get an explicit team decision + ADR before adopting a paid vendor.

#### MCP and assistant behavior

- MCP tools must declare purpose, inputs, outputs, and error behavior.
- Conversation memory: store only what is needed; support deletion/export if user data APIs exist.
- Prefer calling backend services through controlled tools rather than giving the model raw DB credentials.
- When in doubt, fail closed (refuse the tool call) rather than expose sensitive data.
- Tool implementations that need an LLM must call through the Ollama/Grok abstraction—not ad-hoc HTTP to other model vendors.

#### Mandatory: new features get MCP tools

**Rule: when you add a user-facing or agent-useful product feature (API endpoint group, analytics capability, watchlist action, alert type, portfolio view, etc.), you must expose it as an MCP tool in the same MR/PR so the AI assistant can use it.**

| Requirement | Detail |
|---|---|
| Where | Register tools in `backend/internal/transport/mcp` (and `backend/cmd/mcp`); keep names/schemas stable |
| Contract | Prefer calling existing N-layered services or the public HTTP API — do not reimplement business logic in the MCP adapter |
| Python | If the assistant needs a first-class tool binding, update `ai/src/swyngora_ai/tools/` (HTTP mirror and/or MCP client) in the same change |
| Docs | Document the tool in `backend/internal/transport/mcp/README.md` and `docs/features/ai-assistant.md` (or the feature doc) |
| Tests | MCP tool behavior tests (mocked API) + AI tool wiring tests when the Python surface changes |
| Exceptions | Pure internals with no agent value (e.g. cache cleanup ticks) may skip MCP — state why in the MR |

Do **not** ship a feature AI users would expect to ask about while leaving it unreachable from tools.

### 6.6 Background runtime: refresh and cache hygiene

**Rule: when the app (or a long-running service) runs in the background, it must refresh stale data and clean or invalidate cache when needed—not serve outdated market state indefinitely.**

Crypto and stock data go stale quickly. Background modes (mobile app backgrounded, web tab inactive then resumed, Telegram bot long-polling, Go workers, AI assistant sessions) must not assume cached values stay valid forever.

#### When to refresh

| Trigger | Expected behavior |
|---|---|
| App / screen returns to foreground | Revalidate critical market data, watchlists, and open signal views if TTL expired |
| Background sync / worker tick | Fetch only what is due; respect rate limits; skip work when nothing is stale |
| Push / alert path about to fire | Refresh the underlying quote or condition before notifying when freshness matters |
| User-initiated pull-to-refresh or explicit reload | Bypass or short-circuit TTL and fetch fresh data |
| Config or auth change (user, exchange selection, API base URL) | Invalidate related caches immediately |

#### When to clean / invalidate cache

- **TTL expired** — drop or revalidate entries; do not return expired market data as if it were live without marking it stale.
- **Memory / storage pressure** — evict least-recently-used or lowest-priority keys; bound cache size.
- **Logout, account switch, or session end** — clear user-scoped cache and sensitive in-memory state.
- **Schema or API contract change** — wipe or migrate caches that would mis-parse under the new shape.
- **Failed validation or corrupt entry** — delete the bad entry; do not loop on poison cache data.
- **Background period longer than the data’s freshness budget** — full or partial cache clean on resume when appropriate (especially prices, volumes, and signal snapshots).

#### Implementation expectations

| Layer | Guidance |
|---|---|
| Web (React) | On visibility/focus regain, revalidate queries (e.g. stale-while-revalidate); clear or limit persisted cache that can go badly out of date |
| Mobile (React Native) | Use AppState (or equivalent): on `active`, refresh due data; on background, pause nonessential polling; purge expired local storage/AsyncStorage cache keys |
| Go backend | Cache with explicit TTL; background goroutines must refresh or expire entries; provide invalidation hooks for admin/user events |
| AI / MCP | Do not answer with market figures from session memory alone if they may be stale—re-fetch via tools when freshness matters |
| Telegram bot | Refresh context used for alerts/commands when the last fetch is older than the configured TTL |

#### Design defaults

- Every cache entry needs a **TTL** (or explicit “invalidate on event”) and a defined owner for cleanup.
- Prefer **stale-while-revalidate** for UI snappiness, but never present expired critical numbers without a stale indicator when the product shows “live” data.
- Background refresh must be **idempotent**, **rate-limited**, and **cancellable** on shutdown.
- Log cache hits/misses/evictions at debug level in development; avoid logging payload secrets.
- Document TTLs and refresh triggers in the package `README.md` when introducing a new cache (§8.1 / §8.2).
- Add tests for expiry, invalidation on resume/logout, and “do not notify on stale condition without refresh” where alerts exist (§6.3).

### 6.7 Backend architecture: always N-layered

**Rule: the Go backend must always use an N-layered (layered) architecture. Do not put business logic in handlers, talk to the database from HTTP adapters, or collapse layers “for speed.”**

Every backend feature is implemented by composing clear layers. Dependencies point **inward** (outer layers may call inner ones; inner layers must not import outer adapters).

#### Required layers

| Layer (outer → inner) | Responsibility | Typical packages (illustrative) |
|---|---|---|
| **Transport / delivery** | HTTP/gRPC/WebSocket handlers, request decoding, status codes, auth middleware wiring | `internal/handler`, `internal/http`, `internal/transport` |
| **Application / use case** | Orchestrate a single use case; transactions at this boundary; no raw SQL or framework types leaking down | `internal/service`, `internal/usecase`, `internal/app` |
| **Domain** | Entities, value objects, domain rules, domain errors; pure business meaning | `internal/domain`, `internal/entity` |
| **Infrastructure / adapters** | DB, cache, external market APIs, message queues, email—implement ports defined by inner layers | `internal/repository`, `internal/infra`, `internal/adapter` |

Optional but encouraged when useful:

| Layer | When |
|---|---|
| **DTO / API models** | Map transport payloads ↔ domain without exposing DB models on the wire |
| **Ports / interfaces** | Define repository and gateway interfaces in application or domain; implement them in infrastructure |

Exact folder names may vary; the **separation of concerns** must not.

#### Hard rules

1. **Handlers are thin** — parse/validate input, call one use-case/service method, map result to HTTP. No business rules, no SQL, no direct external API calls.
2. **Use cases own orchestration** — multi-step flows, authorization checks that are business rules, and unit-of-work live here.
3. **Domain stays framework-free** — no `net/http`, SQL drivers, or Gin/Echo/Fiber types inside domain packages.
4. **Infrastructure implements interfaces** — repositories and market-data clients are injected into services; services do not import concrete driver packages when avoidable.
5. **No layer skipping for writes/reads of substance** — e.g. handler → repository is forbidden; go handler → service → repository (or equivalent).
6. **Shared kernel only for truly shared types** — do not create a dumping-ground `models` package that every layer mutates freely.
7. **New endpoints and workers follow the same layering** — background jobs and CLI entrypoints are just another transport; they still call application services.
8. **Tests follow layers** — domain and use-case unit tests with fakes; handler tests with mocked services; repository tests against fakes or test DB (§6.3).

#### Anti-patterns (reject in review)

- “God” handler files that query the DB and call exchanges.
- Domain entities tagged with ORM/JSON transport concerns as the only model.
- Circular imports between `handler` ↔ `repository` or `service` ↔ `http`.
- Copy-pasting business rules into multiple handlers instead of a use case.
- New micro-packages that bypass layering without an ADR.

#### Skeleton (target shape under `backend/`)

```text
backend/
├── cmd/                 # main entrypoints only (wire dependencies)
├── internal/
│   ├── transport/       # or handler/http — delivery layer
│   ├── service/         # or usecase/app — application layer
│   ├── domain/          # entities, domain logic
│   ├── repository/      # or adapter/infra — infrastructure
│   └── platform/        # config, logging, shared middleware wiring
├── api/
│   └── openapi/         # OpenAPI specs (source of truth for HTTP API — §6.9)
└── README.md            # how layers map in this service
```

When scaffolding or extending the backend, preserve this structure and document deviations in `backend/README.md` and this file (§8.2). Any intentional change to the layering model requires an ADR under `docs/adr/`.

### 6.8 Client architecture: Atomic Design and file separation

**Rule: React web (`web/`) and React Native (`mobile/`) must follow Atomic Design for UI components, and must separate types, constants, and helpers into dedicated files—not dump everything into one component file.**

Applies to all client UI code. Telegram bot is excluded unless it gains a shared React surface.

#### Atomic Design hierarchy

Organize components from smallest (generic) to largest (page-level):

| Level | Role | Examples |
|---|---|---|
| **Atoms** | Smallest UI primitives; no feature domain knowledge | `Button`, `Input`, `Text`, `Icon`, `Spinner` |
| **Molecules** | Simple combinations of atoms | `SearchField`, `LabelValue`, `IconButton` |
| **Organisms** | Relatively complex, reusable sections | `WatchlistTable`, `PriceChartHeader`, `AlertForm` |
| **Templates** | Page-level layout shells without real data wiring | `DashboardTemplate`, `AuthTemplate` |
| **Pages** (or screens on mobile) | Route/screen entry: compose templates/organisms and connect data | `WatchlistPage`, `CoinDetailScreen` |

**Dependency direction:** Pages → Templates → Organisms → Molecules → Atoms. Lower levels must not import higher levels (no atom importing an organism).

#### Target folder shape (per client app)

```text
src/
├── components/                 # or ui/
│   ├── atoms/
│   │   └── Button/
│   │       ├── Button.tsx
│   │       ├── Button.types.ts
│   │       ├── constants.ts    # if the atom needs local constants
│   │       ├── helpers.ts      # if the atom needs pure helpers
│   │       ├── Button.test.tsx
│   │       └── index.ts        # public export barrel (optional but preferred)
│   ├── molecules/
│   ├── organisms/
│   ├── templates/
│   └── pages/                  # or screens/ on React Native
├── features/                   # optional: feature-scoped composition that still uses Atomic levels
├── hooks/
├── services/                   # API clients — not UI atoms
└── ...
```

- Shared design-system primitives live under `atoms` / `molecules`.
- Feature-specific UI that is still reusable within a feature may live under `features/<name>/components/` **using the same Atomic levels and file rules**.
- Do not place business/API fetching inside atoms or molecules; data loading belongs in pages/screens, hooks, or feature containers that feed organisms via props.

#### Mandatory file separation

Within a component folder (or module folder), split concerns into dedicated files:

| Concern | File name | Contents |
|---|---|---|
| Types | `*.types.ts` or `<Name>.types.ts` | Props, view models, unions, mapped API types used by that unit |
| Constants | `constants.ts` | Magic numbers/strings, query keys local to the unit, enum-like maps, layout tokens |
| Helpers | `helpers.ts` | Pure functions (formatters, guards, mappers) with no React component definitions |
| Component | `<Name>.tsx` | JSX/component only—imports types, constants, and helpers |
| Tests | `<Name>.test.tsx` (or colocated `__tests__`) | Behavior tests for the unit |
| Barrel | `index.ts` | Re-export public API of the folder |

**Naming rules**

- Types: always suffix with `.types.ts` (e.g. `WatchlistTable.types.ts`, `priceChart.types.ts`). Never bury exported interfaces/types only inside the `.tsx` when they are shared or non-trivial—prefer `.types.ts`.
- Constants: file is always named **`constants.ts`** (not `const.ts`, not `contants.ts`).
- Helpers: file is always named **`helpers.ts`** (not `utils.ts` inside a component folder—use `helpers.ts` for consistency). App-wide shared pure utils for the product web app live under **`frontend/src/libs/utils/`** (see package `AGENTS.md`); component-local pure helpers stay as `helpers.ts`.
- One primary component per folder when practical; name the folder after the component.

**What belongs where**

```text
// WatchlistTable.types.ts  — types only
// constants.ts             — e.g. COLUMN_IDS, DEFAULT_PAGE_SIZE
// helpers.ts               — e.g. sortByMarketCap(), formatVolume()
// WatchlistTable.tsx       — component composition + hooks usage
```

- Trivial one-liner local types used once may stay in the `.tsx` only if they are not exported; prefer `.types.ts` as soon as props or models grow.
- Do not put React components inside `helpers.ts` or `constants.ts`.
- Do not put mutable module state in `constants.ts`.
- Shared cross-feature types/constants/helpers live in clearly named shared folders and still use the same suffixes/filenames.

#### Hard rules

1. **Always Atomic Design** for new UI—no flat `components/` dumping ground of unrelated mixed-level files.
2. **Always separate** non-trivial types → `.types.ts`, constants → `constants.ts`, helpers → `helpers.ts`.
3. **No upward imports** across Atomic levels.
4. **Web and mobile share the same conventions**; platform-specific components get platform folders or `.native.tsx` / `.web.tsx` only when required, still under the correct Atomic level.
5. When adding a new atoms/molecules package tree, document it in the package `README.md` (§8.1 / §8.2).
6. **Product web (`frontend/`)** standard UI kit is **Ant Design**; financial charts use **TradingView Lightweight Charts**. Do not add a second UI kit or chart library without a decision under `project-management/decisions/` (or ADR). Prefer wrapping `antd` in Atomic components. Shared API/hooks/utils live in **`frontend/src/libs/`**. Local task board: **`project-management/`**.

#### Anti-patterns (reject in review)

- 500-line `.tsx` files with types, constants, helpers, and JSX all inline.
- `utils.ts` / `types.ts` without the required names (`helpers.ts`, `*.types.ts`, `constants.ts`) inside component modules.
- Atoms that fetch from the network or know exchange-specific business workflows (use RTK Query at page/feature layer — §6.9).
- Pages imported by organisms/molecules.
- Copy-pasting the same helper into multiple components instead of a shared `helpers.ts`.

### 6.9 Client data layer: RTK Query + OpenAPI-generated types

**Rule: React web and React Native use RTK Query (Redux Toolkit Query) for server/async API state. Backend public HTTP types for clients are produced from OpenAPI—do not hand-maintain duplicate request/response TypeScript models.**

#### OpenAPI is the contract

| Concern | Rule |
|---|---|
| Source of truth | OpenAPI spec(s) under `backend/api/openapi/` (or path documented in `backend/README.md`) |
| Backend | Handlers and DTOs must match the spec; change the spec in the **same MR** as breaking/additive API changes |
| Client types | **Generated** from OpenAPI (e.g. `openapi-typescript`, `@rtk-query/codegen-openapi`, or equivalent). Never copy-paste API shapes into hand-written types as the long-term source |
| Versioning | Spec changes follow SemVer impact (§4); breaking schema changes are MAJOR |

**Workflow for any new or changed public endpoint**

1. Update the OpenAPI spec (paths, schemas, examples, auth).
2. Implement backend to satisfy the spec (N-layered — §6.7).
3. Regenerate client types / RTK Query API definitions.
4. Wire UI via generated hooks; update Atomic Design components as needed (§6.8).
5. Refresh docs (`docs/api/`, package READMEs) per §8.1 / §8.2.
6. Add/adjust tests for backend and client usage (§6.3).

Do not land a client feature that talks to a backend route **without** an OpenAPI description and regenerated types.

#### RTK Query (mandatory for client ↔ backend HTTP)

| Do | Don't |
|---|---|
| Define APIs with `createApi` / `injectEndpoints` (or codegen output) | Scatter raw `fetch` / `axios` calls across components for backend REST |
| Use generated or hand-written endpoints that consume **OpenAPI-derived types** | Hand-roll parallel TS interfaces that drift from the spec |
| Prefer cache tags, `providesTags` / `invalidatesTags`, and fixed cache lifetimes aligned with §6.6 | Infinite stale market data with no invalidation |
| Colocate domain endpoints under `src/libs/api/endpoints/` (product web) | Put RTK Query endpoint definitions inside atoms/molecules or `features/*/api` for backend REST |
| Use RTK Query hooks in pages, screens, or feature containers; pass data down as props | Import `useGetXQuery` deep inside pure presentational atoms |

**Suggested client layout (product web `frontend/` — libs-first)**

```text
src/
├── libs/
│   ├── api/
│   │   ├── baseApi.ts        # createApi base (baseUrl, auth headers)
│   │   ├── store.ts          # configureStore — API middleware & reducer
│   │   ├── hooks.ts          # typed useAppDispatch / useAppSelector
│   │   ├── endpoints/        # injectEndpoints per domain (marketApi, …)
│   │   ├── generated/        # OpenAPI codegen output (DO NOT hand-edit)
│   │   └── index.ts
│   ├── hooks/                # shared React hooks (visibility, debounce, …)
│   ├── utils/                # pure helpers (formatters, query builders)
│   └── types/                # optional re-exports / view types (not hand DTOs)
├── components/               # Atomic Design UI
├── features/                 # feature UI only (no backend REST modules)
└── ...
```

- Product frontend: **API + shared hooks + utils live under `src/libs/`** — do not put backend REST under `features/*/api`.
- Treat `libs/api/generated/` (or equivalent) as **read-only**; fix the OpenAPI spec and re-run codegen instead of patching generated files.
- Map OpenAPI schemas into UI-only view models in `*.types.ts` / `helpers.ts` when the screen needs a different shape—do not “fix” the generated API types in place.
- Background refresh and cache hygiene (§6.6) must use RTK Query invalidation, refetch-on-focus, and polling options where appropriate—not a second ad-hoc cache.

#### Codegen expectations

- Document the exact generator and command in `web/README.md` and `mobile/README.md` once scaffolded (e.g. `npm run codegen:api`).
- CI should fail (or warn loudly) when the OpenAPI spec changes but generated client artifacts are not updated—prefer a check script once tooling exists.
- Shared monorepo package for generated types is allowed (e.g. `packages/api-types`) if web and mobile should consume one artifact; still generate from the same OpenAPI source.

#### Anti-patterns (reject in review)

- Hand-written duplicate DTO interfaces that mirror the backend without codegen.
- Editing files under `generated/` by hand.
- `useEffect` + manual `fetch` for standard CRUD/list/detail backend calls when an RTK Query endpoint should exist.
- Shipping API changes without updating OpenAPI.
- Importing generated API types into Go domain packages (wrong direction)—OpenAPI drives **client** types and documents the **HTTP** boundary; Go domain stays independent (§6.7).

---

## 7. Tooling and commands (evolve as the monorepo grows)

Until packages exist, use these as the intended conventions. **Update this section when real scripts land.**

```bash
# Backend (Go) — from backend/
go test ./...
go vet ./...
golangci-lint run   # when configured
# keep OpenAPI in sync with handlers (lint/validate when tooling exists)
# swagger-cli validate api/openapi/*.yaml   # example

# AI (Python) — from ai/
python3 -m venv .venv && source .venv/bin/activate
pip install -e ".[dev]"
pytest -q
ruff check . && ruff format --check .
# swyngora-ai "What is BTC RSI on binance?"   # needs Ollama or XAI_API_KEY + API up

# Backend (implemented) — from backend/
go test ./...
go run ./cmd/server   # :8080 REST + /mcp (streamable MCP) in one process
# go run ./cmd/mcp    # optional stdio-only MCP (not required)
go test ./internal/transport/mcp/...

# Telegram bot is integrated in backend (optional): set TELEGRAM_BOT_TOKEN then go run ./cmd/server
# go test ./internal/transport/telegram/...

# Simple frontend (test harness) — from simple-frontend/
python3 -m http.server 5173   # open http://localhost:5173

# Product frontend — from frontend/ (when scaffolded)
# npm run codegen:api
# npm test && npm run lint && npm run build

# Mobile (React Native + react-native-web) — from mobile/
# npm install
# npm run web           # Chrome → http://localhost:5180 (primary; no simulator required)
# npm test && npm run lint && npm run typecheck
# npm run codegen:api   # same OpenAPI source as frontend
```

CI should run the relevant subset of the above on every MR/PR to `develop` and `main`.

---

## 8. Documentation

| Doc | Audience | Purpose |
|---|---|---|
| `README.md` | Humans | Quick start, vision, links |
| `AGENTS.md` | Agents + team | Workflow, versioning, engineering defaults |
| Package / folder `README.md` | Humans + agents | How to build, run, test, and extend that area |
| `docs/` | Everyone | ADRs, architecture diagrams, API notes, feature guides |
| `CHANGELOG.md` | Everyone | Release history |

When you make a durable architectural decision (e.g. choice of market-data provider, auth model), add a short ADR under `docs/adr/` rather than only mentioning it in chat.

### 8.1 Documentation required for new folders and features

**Rule: introducing a new top-level or significant package folder, or shipping a new feature, requires documentation in the same MR/PR as the code.**

#### New folder / package

When you create a new directory that owns a subsystem (e.g. `backend/internal/marketdata/`, `ai/tools/`, `web/src/features/watchlist/`), add a **`README.md` inside that folder** (or update the parent package README if the folder is tiny and clearly owned there). Minimum contents:

1. **Purpose** — what this folder is for (1–3 sentences).
2. **Layout** — key files/subpackages and responsibilities.
3. **How to run / test** — commands from that package’s perspective.
4. **Dependencies** — internal packages and external services it talks to.
5. **Config / env** — required variables (names only; never real secrets).
6. **Ownership notes** — anything a newcomer would otherwise have to ask in chat.

Also update the root `README.md` (and §2 of this file if the top-level layout changed) so discovery stays accurate.

#### New feature

When you ship a new user-facing or cross-cutting capability (API endpoint group, indicator suite, paper trading flow, MCP tool set, bot command group, etc.), document it in the same change set:

| Surface | Document |
|---|---|
| Backend API / protocol | **OpenAPI** updated in-repo; regenerate client types; optional `docs/api/` note; auth requirements (§6.9) |
| AI / MCP tool | Tool name, purpose, input/output schema, failure modes; short usage example |
| Web / mobile UI | Brief feature note in package docs or `docs/features/<name>.md` (what it does, main screens/routes) |
| Telegram bot | Commands, args, and example dialogs |
| Cross-cutting design choice | Short ADR in `docs/adr/` when the decision is durable |

**Feature doc minimum (when adding `docs/features/<name>.md` or equivalent)**

1. Problem / goal.
2. Behavior (happy path + important limits).
3. Where the code lives (paths).
4. How to test / verify.
5. Known limitations or follow-ups.

#### What does *not* need a full feature doc

- Tiny bug fixes with no new surface area (still add a changelog entry if user-visible).
- Pure refactors that preserve behavior (optional short note if structure moved).
- One-line config tweaks.

Stale or missing docs for new structure are treated like missing tests: the MR is not done.

### 8.2 Keep `AGENTS.md` and `README.md` in sync with code

**Rule: after changing any code, update every related `AGENTS.md` and `README.md` in the same MR/PR. Do not leave agent or human docs describing old layout, commands, APIs, or conventions.**

Whenever you edit production or tooling code, ask: *did this make any README or AGENTS file wrong or incomplete?* If yes, fix those files before finishing.

| What changed | Update at least |
|---|---|
| New/removed package or top-level folder | Root `README.md`, root `AGENTS.md` §2 (layout); package `README.md` / nested `AGENTS.md` |
| Build, test, lint, or run commands | Package `README.md` and root/package `AGENTS.md` tooling sections (§7 or local equivalent) |
| Stack, providers, env vars, or defaults (e.g. Ollama/Grok, ports) | Root and package `AGENTS.md` + relevant `README.md` config sections |
| Public HTTP API (OpenAPI) | Spec under `backend/api/openapi/`; regenerate client types; `web`/`mobile` README codegen notes (§6.9) |
| Public API, MCP tools, bot commands, or feature behavior | Feature/API docs (§8.1) **and** any README that describes that surface |
| Git Flow, versioning, test, or doc process | Root `AGENTS.md` only (plus root `README.md` if it summarizes contributing) |
| Nested package conventions | Nearest `AGENTS.md` (closest wins) and that package’s `README.md` |

**How to apply**

1. Prefer updating docs **in the same commit** as the code, or in the same branch before merge—never “docs later.”
2. Touch only the docs that the change affects; do not rewrite unrelated sections.
3. If a code change has **no** doc impact (e.g. pure bug fix with identical public behavior and commands), state that briefly in the MR—still re-read the nearest README/`AGENTS.md` to confirm.
4. Nested package: update the **closest** `AGENTS.md` and that package’s `README.md`; update the root files only when the change is visible at monorepo level.
5. Agents must treat stale `AGENTS.md` / `README.md` as a defect in the change set, same severity as missing tests for a new feature.

**Checklist before marking work done**

- [ ] Grep or skim for outdated paths, commands, env names, and provider names you just changed.
- [ ] Root and/or package `README.md` still match how to build, run, and test.
- [ ] Root and/or nested `AGENTS.md` still match real conventions and layout.
- [ ] New folders/features also satisfy §8.1.

---

## 9. Issue and task hygiene

- Link MRs/PRs to issues when the tracker is used (GitLab issues on this remote).
- Prefer vertical slices: end-to-end thin features over long-lived half-integrated layers.
- Label work by area: `backend`, `ai`, `web`, `mobile`, `bot`, `infra`, `security`.

---

## 10. What agents must not do

- Force-push or rewrite history on `main` / `develop`.
- Publish shared work to only one of `origin` / `beratersari` when both remotes are configured (§3.8).
- Merge to `main` without going through release or hotfix flow.
- Invent production credentials or disable auth “temporarily” in committed code.
- Add heavy dependencies without a clear need.
- Create large unrelated file trees “for later.”
- Skip updating `VERSION` / `CHANGELOG.md` / git tag when performing a release task.
- Land a **feature or new production file without tests** (see §6.3).
- Introduce a **new subsystem folder or feature without documentation** (see §8.1).
- Change code and **leave related `AGENTS.md` or `README.md` stale** (see §8.2).
- Run long-lived or background paths that **never refresh or clean cache**, serving unbounded stale market data (§6.6).
- Implement backend features **outside N-layered architecture** (e.g. business logic or DB access in handlers) (§6.7).
- Build client UI **outside Atomic Design**, or mix types/constants/helpers into large `.tsx` files instead of `*.types.ts` / `constants.ts` / `helpers.ts` (§6.8).
- Call the backend from clients **without RTK Query**, or maintain **hand-written API DTOs** instead of **OpenAPI-generated** types (§6.9).
- Change public HTTP APIs **without updating OpenAPI** and regenerating clients (§6.9).
- Wire **non-Ollama / non-Grok** commercial LLM providers as defaults or “pricing options.”
- Add external data-API integrations that require **paid pricing-tier choices** without an ADR and team approval.
- Ship a **new agent-useful feature without an MCP tool** (and Python tool binding when applicable) — see §6.5 “Mandatory: new features get MCP tools.”

---

## 11. Quick reference card

```text
Feature work:   develop → feature/* → MR → develop
Release:        develop → release/vX.Y.Z → main + tag → back-merge develop
Hotfix:         main → hotfix/vX.Y.Z → main + tag → back-merge develop
Versioning:     SemVer MAJOR.MINOR.PATCH · tags vX.Y.Z · VERSION + CHANGELOG
Commits + MR titles: Conventional Commits · type(scope): subject (§5.1)
Tests:          Required for every feature + new/changed production logic (same MR)
Docs:           Folder README for new packages; feature/API/ADR docs for new capabilities
After code:     Update related AGENTS.md + README.md in the same MR (§8.2)
Background:     Refresh stale data + clean/invalidate cache when needed (§6.6)
Backend:        Always N-layered (transport → application → domain ← infrastructure) (§6.7)
Client UI:      Atomic Design; product web: Ant Design + Lightweight Charts; libs/ for api·hooks·utils (§6.8)
Client data:    RTK Query + types/endpoints from OpenAPI codegen (§6.9)
Local PM:       project-management/ (epics, tasks, board) until GitLab fully used
LLMs:           Local Ollama + Grok (xAI) only — no other commercial LLM defaults
AI:             ai/ LangGraph orchestrator + specialists; backend/cmd/mcp tools
New features:   Add MCP tool (+ AI tool binding) in the same MR when agent-useful (§6.5)
External APIs:  No paid pricing-tier choices; prefer free/public/self-hosted data sources
Default branch for integration: develop
Production branch: main
Remotes:        origin = GitLab (primary); beratersari = GitHub private mirror
Push:           git pushboth <ref>  # both remotes (§3.8)
```

---

## 12. Living document

This project is early. When stack choices solidify (module paths, package managers, CI jobs, deploy targets), update **§2**, **§7**, nested package `AGENTS.md`, and related `README.md` files in the same change set (see §8.2). Stale agent docs are worse than short ones.

**Last updated:** 2026-08-09 (`ai/`: Ruff lint + format; E/W/F/I/UP)  
**Initial product version target:** `0.1.0` (pre-release development)
