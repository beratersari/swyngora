#!/usr/bin/env bash
# Create Frontend epics and issues on GitLab (origin project).
# Usage:
#   export GITLAB_TOKEN="glpat-..."   # api scope
#   export GITLAB_HOST="https://nova.teachx.ai"
#   export GITLAB_PROJECT="trace-analysis/swyngora"
#   ./docs/pm/create-gitlab-epics.sh
#
# Notes:
# - Epics require GitLab Premium/Ultimate group epics API on many instances.
# - If epics API is unavailable, the script creates a parent issue + child issues linked in description.
set -euo pipefail

HOST="${GITLAB_HOST:-https://nova.teachx.ai}"
PROJECT="${GITLAB_PROJECT:-trace-analysis/swyngora}"
TOKEN="${GITLAB_TOKEN:-${GITLAB_PERSONAL_ACCESS_TOKEN:-}}"

if [[ -z "${TOKEN}" ]]; then
  echo "error: set GITLAB_TOKEN (Personal Access Token with api scope)" >&2
  exit 1
fi

API="${HOST%/}/api/v4"
PROJ_ENC=$(python3 -c "import urllib.parse; print(urllib.parse.quote('${PROJECT}', safe=''))")
AUTH=(--header "PRIVATE-TOKEN: ${TOKEN}" --header "Content-Type: application/json")

echo "Project: ${PROJECT} @ ${HOST}"

# Resolve project id + namespace (for epics)
PROJECT_JSON=$(curl -sS "${AUTH[@]}" "${API}/projects/${PROJ_ENC}")
PROJECT_ID=$(echo "${PROJECT_JSON}" | python3 -c "import sys,json; print(json.load(sys.stdin)['id'])")
NAMESPACE_ID=$(echo "${PROJECT_JSON}" | python3 -c "import sys,json; print(json.load(sys.stdin)['namespace']['id'])")
echo "project_id=${PROJECT_ID} namespace_id=${NAMESPACE_ID}"

create_issue() {
  local title="$1"
  local description="$2"
  local labels="$3"
  curl -sS "${AUTH[@]}" -X POST "${API}/projects/${PROJECT_ID}/issues" \
    --data-binary @- <<JSON
{"title": $(python3 -c 'import json,sys; print(json.dumps(sys.argv[1]))' "$title"),
 "description": $(python3 -c 'import json,sys; print(json.dumps(sys.argv[1]))' "$description"),
 "labels": $(python3 -c 'import json,sys; print(json.dumps(sys.argv[1]))' "$labels")}
JSON
}

create_epic() {
  local title="$1"
  local description="$2"
  local labels="$3"
  # Group epics API
  curl -sS -w "\n%{http_code}" "${AUTH[@]}" -X POST "${API}/groups/${NAMESPACE_ID}/epics" \
    --data-binary @- <<JSON
{"title": $(python3 -c 'import json,sys; print(json.dumps(sys.argv[1]))' "$title"),
 "description": $(python3 -c 'import json,sys; print(json.dumps(sys.argv[1]))' "$description"),
 "labels": $(python3 -c 'import json,sys; print(json.dumps(sys.argv[1]))' "$labels")}
JSON
}

echo "--- Epic 1: Frontend project initialization ---"
EPIC1_DESC=$(cat <<'MD'
## Summary

Initialize the production React web app under `frontend/` so feature work can start.

This is **P0** and **blocks** multi-exchange spot markets UI.

## Design docs

- `docs/design/frontend-project-initialization.md`
- `docs/design/frontend-system-design.md`
- `docs/pm/frontend-epics-and-issues.md`

## Acceptance

- `npm install && npm run dev` works
- `npm run codegen:api` works
- Placeholder Markets route renders
- Structure matches AGENTS.md §6.8 / §6.9
MD
)

EPIC1_RESP=$(create_epic "[frontend] Project initialization" "$EPIC1_DESC" "frontend,priority::p0" || true)
EPIC1_CODE=$(echo "$EPIC1_RESP" | tail -n1)
EPIC1_BODY=$(echo "$EPIC1_RESP" | sed '$d')
EPIC_MODE="epic"
if [[ "$EPIC1_CODE" != "201" && "$EPIC1_CODE" != "200" ]]; then
  echo "warn: group epics API returned HTTP ${EPIC1_CODE}; falling back to parent issue"
  echo "$EPIC1_BODY" | head -c 400; echo
  EPIC_MODE="issue"
  PARENT1=$(create_issue "[Epic][frontend] Project initialization" "$EPIC1_DESC" "frontend,priority::p0,type::epic")
  PARENT1_IID=$(echo "$PARENT1" | python3 -c "import sys,json; print(json.load(sys.stdin)['iid'])")
  echo "Created parent issue !${PARENT1_IID}"
else
  EPIC1_IID=$(echo "$EPIC1_BODY" | python3 -c "import sys,json; print(json.load(sys.stdin).get('iid') or json.load(sys.stdin).get('id'))")
  echo "Created epic &${EPIC1_IID} (or id)"
fi

create_init_issue() {
  local key="$1" title="$2" body="$3"
  local desc
  desc=$(printf '%s\n\n## Parent\nEpic 1: Frontend project initialization\n\n## Key\n`%s`\n\n%s\n' \
    "Part of **Frontend project initialization**." "$key" "$body")
  local res iid
  res=$(create_issue "[frontend] ${title}" "$desc" "frontend,type::chore,priority::p0")
  iid=$(echo "$res" | python3 -c "import sys,json; print(json.load(sys.stdin)['iid'])")
  web=$(echo "$res" | python3 -c "import sys,json; print(json.load(sys.stdin).get('web_url',''))")
  echo "  ${key} → #${iid} ${web}"
}

create_init_issue "INIT-1" "Scaffold Vite React TypeScript app" \
  "Create Vite+React+TS under frontend/. See docs/design/frontend-project-initialization.md"
create_init_issue "INIT-2" "Add lint, format, Vitest, path aliases" \
  "Depends on INIT-1. ESLint/Prettier/Vitest and @/ alias."
create_init_issue "INIT-3" "Atomic Design + feature folder skeleton" \
  "Align frontend/src tree with system design. Depends on INIT-1."
create_init_issue "INIT-4" "Wire Redux store + RTK Query baseApi" \
  "configureStore + baseApi + typed hooks. Depends on INIT-1/2."
create_init_issue "INIT-5" "OpenAPI codegen pipeline" \
  "npm run codegen:api from backend OpenAPI. Depends on INIT-4."
create_init_issue "INIT-6" "App shell, router, env, Markets placeholder" \
  "Router + VITE_API_BASE_URL + placeholder Markets route. Depends on INIT-4."
create_init_issue "INIT-7" "Docs: frontend README + AGENTS.md" \
  "Package docs + root links + changelog if needed. Depends on INIT-5/6."
create_init_issue "INIT-8" "Ant Design provider + theme" \
  "Install antd; ConfigProvider; theme tokens. Depends on INIT-1."
create_init_issue "INIT-9" "lightweight-charts + chart wrapper stub" \
  "Install lightweight-charts; chart host stub. Depends on INIT-1/3."

echo "--- Epic 2: Multi-exchange spot markets ---"
EPIC2_DESC=$(cat <<'MD'
## Summary

Browse spot markets on Binance, Coinbase, and Bybit with search, filters, sort, pagination, and live refresh.

## Backend APIs (ready)

- GET /api/v1/market/exchanges
- GET /api/v1/market/tags
- GET /api/v1/market/spot

## Design

- docs/features/multi-exchange-spot-markets.md

## Blocked by

Epic: Frontend project initialization
MD
)

if [[ "$EPIC_MODE" == "epic" ]]; then
  EPIC2_RESP=$(create_epic "[frontend] Multi-exchange spot markets" "$EPIC2_DESC" "frontend,type::feature,priority::p1" || true)
  echo "$EPIC2_RESP" | tail -n3
else
  PARENT2=$(create_issue "[Epic][frontend] Multi-exchange spot markets" "$EPIC2_DESC" "frontend,priority::p1,type::epic,type::feature")
  echo "$PARENT2" | python3 -c "import sys,json; d=json.load(sys.stdin); print('parent', d['iid'], d.get('web_url',''))"
fi

create_mkt_issue() {
  local key="$1" title="$2" body="$3"
  local desc
  desc=$(printf '%s\n\n## Parent\nEpic 2: Multi-exchange spot markets\n\n## Key\n`%s`\n\n## Blocked by\nEpic 1 (frontend project initialization)\n\n%s\n' \
    "Part of **Multi-exchange spot markets**." "$key" "$body")
  local res iid web
  res=$(create_issue "[frontend] ${title}" "$desc" "frontend,type::feature,priority::p1")
  iid=$(echo "$res" | python3 -c "import sys,json; print(json.load(sys.stdin)['iid'])")
  web=$(echo "$res" | python3 -c "import sys,json; print(json.load(sys.stdin).get('web_url',''))")
  echo "  ${key} → #${iid} ${web}"
}

create_mkt_issue "MKT-1" "RTK market endpoints (exchanges, tags, spot)" \
  "Wire OpenAPI-typed RTK endpoints for multi-exchange spot."
create_mkt_issue "MKT-2" "Markets page shell + ExchangeTabs" \
  "Depends on MKT-1."
create_mkt_issue "MKT-3" "MarketsTable + column formatters" \
  "Depends on MKT-2."
create_mkt_issue "MKT-4" "Toolbar: search, quote, tags, sort" \
  "Depends on MKT-3."
create_mkt_issue "MKT-5" "Pagination + URL query sync" \
  "Depends on MKT-4."
create_mkt_issue "MKT-6" "Live poll + visibility pause" \
  "Depends on MKT-3."
create_mkt_issue "MKT-7" "Empty/error UX + tests" \
  "Depends on MKT-4 and MKT-6."

echo "Done. Review issues on ${HOST}/${PROJECT}/-/issues"
