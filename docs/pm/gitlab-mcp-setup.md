# Configure GitLab project management MCP (Grok)

How to connect Grok Build to **GitLab** so agents can create/list issues, epics, and MRs.

Swyngora primary remote: **https://nova.teachx.ai** · project `trace-analysis/swyngora`.

---

## Option A — Official GitLab MCP (recommended when available)

GitLab ships a built-in MCP server at:

```text
https://<your-gitlab-host>/api/v4/mcp
```

### Prerequisites (instance admin / group)

On GitLab Self-Managed (`nova.teachx.ai`), an admin must:

1. Enable **GitLab Duo** availability (instance or top-level group) as required by your version.
2. Allow **MCP server** access (Admin → Settings → visibility / Duo / MCP — exact menu depends on GitLab version).
3. Confirm the endpoint responds (not 404):

```bash
curl -sS -o /dev/null -w "%{http_code}\n" "https://nova.teachx.ai/api/v4/mcp"
# Expect something other than 404 when enabled (often 401 without auth)
```

As of the last check in this repo’s setup notes, `nova.teachx.ai/api/v4/mcp` returned **404**, meaning the official MCP server is **not enabled** (or not on that version). Use **Option B** until ops enables it.

### Configure Grok (HTTP / OAuth)

**User scope** (`~/.grok/config.toml`):

```toml
[mcp_servers.gitlab]
url = "https://nova.teachx.ai/api/v4/mcp"
enabled = true
```

Or CLI:

```bash
grok mcp add --transport http gitlab https://nova.teachx.ai/api/v4/mcp
```

**Project scope** (optional, commit-safe — no secrets):

```bash
cd /path/to/swyngora
mkdir -p .grok
# add [mcp_servers.gitlab] with url only
grok mcp add --scope project --transport http gitlab https://nova.teachx.ai/api/v4/mcp
```

Then:

1. Restart Grok / open `/mcps`
2. Authenticate the GitLab server (OAuth browser flow — key `i` in MCP modal or follow prompts)
3. Verify tools appear (`create_issue`, `get_issue`, merge request tools, etc. — set depends on GitLab version)

Docs: [GitLab MCP server](https://docs.gitlab.com/user/model_context_protocol/mcp_server/)

---

## Option B — Community GitLab MCP with Personal Access Token (works without Duo MCP)

Use when official `/api/v4/mcp` is unavailable. Example package: **`@zereight/mcp-gitlab`** (PAT-based, self-hosted friendly).

### 1. Create a Personal Access Token

1. Open https://nova.teachx.ai/-/user_settings/personal_access_tokens  
2. Create token with scopes at least: **`api`** (and `write_repository` only if you want MR/branch tools)  
3. Copy the token once — do **not** commit it

### 2. Export env vars (shell profile or session)

```bash
export GITLAB_PERSONAL_ACCESS_TOKEN="glpat-..."
export GITLAB_API_URL="https://nova.teachx.ai/api/v4"
# some servers use:
export GITLAB_URL="https://nova.teachx.ai"
```

### 3. Add MCP server to Grok

```bash
grok mcp add gitlab \
  -e GITLAB_PERSONAL_ACCESS_TOKEN=${GITLAB_PERSONAL_ACCESS_TOKEN} \
  -e GITLAB_API_URL=https://nova.teachx.ai/api/v4 \
  -- npx -y @zereight/mcp-gitlab
```

Prefer referencing the env var name so secrets are not written into config:

In `~/.grok/config.toml` (illustrative — exact env keys depend on the package README):

```toml
[mcp_servers.gitlab]
command = "npx"
args = ["-y", "@zereight/mcp-gitlab"]
env = { GITLAB_PERSONAL_ACCESS_TOKEN = "${GITLAB_PERSONAL_ACCESS_TOKEN}", GITLAB_API_URL = "https://nova.teachx.ai/api/v4" }
enabled = true
```

> Put **tokens only** in user-level config or environment — never commit PATs into `.grok/config.toml` in the repo.

### 4. Verify

```bash
grok mcp list
grok mcp doctor gitlab
```

In Grok TUI: `/mcps` → GitLab connected → tools listed.

### 5. Use with Swyngora

After tools are available, ask the agent to:

- Create epic **`[frontend] Project initialization`**
- Create child issues INIT-1…INIT-7  
- Create epic **`[frontend] Multi-exchange spot markets`** + MKT-1…MKT-7  

Issue text source of truth: `docs/pm/frontend-epics-and-issues.md`

---

## Option C — No MCP (API script)

If MCP cannot be enabled yet:

```bash
export GITLAB_TOKEN="glpat-..."
export GITLAB_HOST="https://nova.teachx.ai"
export GITLAB_PROJECT="trace-analysis/swyngora"
./docs/pm/create-gitlab-epics.sh
```

Creates the same epic/issue set via REST.

---

## Grok MCP management cheatsheet

```bash
grok mcp list
grok mcp list --json
grok mcp doctor
grok mcp doctor gitlab
grok mcp remove gitlab          # add --scope user|project if needed
```

In TUI:

| Action | How |
|---|---|
| List servers | `/mcps` |
| Auth OAuth server | `i` on server row |
| Refresh after config edit | `r` |
| Enable/disable | `Space` |

Config locations:

| Scope | File |
|---|---|
| User | `~/.grok/config.toml` |
| Project | `<repo>/.grok/config.toml` |

---

## Security

- Never commit `glpat-` tokens or paste them into tracked files.
- Prefer env vars + user-scope MCP config.
- Least privilege: `api` for issues/epics; add write scopes only if needed.
- Private GitHub mirror (`beratersari`) is separate — open MRs/issues on **GitLab `origin`**.

---

## Troubleshooting

| Symptom | Fix |
|---|---|
| `url .../api/v4/mcp` → 404 | Official MCP not enabled; use Option B or C; ask admin to enable GitLab MCP |
| OAuth fails on self-hosted | Check Duo/MCP settings, HTTPS, allowed redirect URIs |
| Tools missing after add | `grok mcp doctor`; restart session; confirm `enabled = true` |
| 401 from API | Token expired/revoked or missing `api` scope |
| Epics API 403/404 | Group epics need Premium/Ultimate on many installs; script falls back to parent issues |

**Last updated:** 2026-07-26
