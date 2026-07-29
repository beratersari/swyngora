# MCAT-A: Tags + spot tag-filter field matrix (analysis)

| Field | Value |
|---|---|
| **ID** | MCAT-A |
| **Epic** | mobile-category-discovery |
| **Status** | done |
| **Area** | mobile / analysis |
| **Path** | `project-management/tasks/mobile/category-discovery/MCAT-A.md` |

## Purpose

Map backend **product tags** and **spot tag filters** to mobile category discovery. **No backend work required** — APIs exist.

Sources:

- OpenAPI: `backend/api/openapi/openapi.yaml` (`listProductTags`, `listSpotMarkets`)
- Feature note: `docs/features/market-data.md` (tags, tag filter, cross-venue enrichment)
- Mobile today: `MarketsContext.selectedTags`, `MarketsFilterPage` + `useListProductTagsQuery`, `spotQuery` / markets RTK

---

## 1. Endpoints

| Method | Path | operationId | Mobile use |
|--------|------|-------------|------------|
| `GET` | `/api/v1/market/tags` | `listProductTags` | Categories browse + featured chips |
| `GET` | `/api/v1/market/spot` | `listSpotMarkets` | Tag-filtered market list (reuse MarketsPage) |

Both read-only. Tags list is empty for non-Binance `exchange` query on `/tags` (catalog source is Binance).

---

## 2. List tags — `GET /api/v1/market/tags`

### Query

| Param | Required | Default | UI |
|-------|----------|---------|-----|
| `exchange` | no | `binance` | Discovery should request **binance** for a non-empty catalog |

### Response

| Field | Type | UI |
|-------|------|-----|
| `exchange` | string | Meta / debug |
| `tags` | `string[]` | Sorted unique labels (e.g. `AI`, `Layer1_Layer2`, `Meme`, `defi`) |

Non-crypto tags (`bStocks`, `tCommodities`) are excluded server-side.

---

## 3. Spot list tag filter — `GET /api/v1/market/spot`

### Tag-related params

| Param | Semantics | UI v1 |
|-------|-----------|-------|
| `tag` | Case-insensitive; comma-separated or repeated = **OR** | Single tag string |
| `tags` | Alias of `tag` | Prefer one param in helpers (`tag`) for consistency with existing mobile spot query |

### Related params to reuse

| Param | Discovery default |
|-------|-------------------|
| `exchange` | Current Markets exchange (or binance default) |
| `quote` | `USDT` (Coinbase may use `USD` — respect Markets quote context) |
| `sort` | `quoteVolume` |
| `order` | `desc` |
| `limit` / `offset` | Existing Markets pagination |

### Response fields for list rows

Reuse existing spot DTO: `symbol`, 24h metrics, `tags[]`, mcap fields, etc. Show `tags` on row if already supported.

---

## 4. Mobile gap matrix

| Capability | Backend | Mobile today | Category epic |
|------------|---------|--------------|---------------|
| List all tags | ✅ `/tags` | Filter page only | Browse page + Home |
| Filter spot by tag | ✅ `tag`/`tags` | Filter form multi-select | Single-tag discovery path |
| Featured subset | ❌ (client constants) | — | Curated ∩ live tags |
| Tag counts | ❌ | — | Out of scope |
| Multi-tag OR browse | ✅ | Filters multi | Keep on Filters only |

---

## 5. Decisions for implementation tasks

1. **Single-tag** apply from discovery → `selectedTags = [tag]`.  
2. **Tags query exchange** for catalog = `binance` always (or configurable constant).  
3. **Results** reuse MarketsPage + context; no second list implementation.  
4. **Featured** constants in `config/categoryConstants.ts`; intersect with live tags.  
5. **No OpenAPI / MCP changes** in this epic.

---

## Acceptance

- [x] Matrix above complete and linked from design/epic  
- [x] Confirmed RTK hooks already exist (`useListProductTagsQuery`, spot list) — no MCAT API layer task unless gap found  
- [x] Status → done when analysis accepted  

## Status

`done`
