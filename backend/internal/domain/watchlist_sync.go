package domain

// ItemsByKey indexes items by ItemKey.
func ItemsByKey(items []WatchlistItem) map[string]WatchlistItem {
	m := make(map[string]WatchlistItem, len(items))
	for _, it := range items {
		m[it.ItemKey()] = it
	}
	return m
}

// MergeWatchlistReplace performs a 3-way merge of base → client vs base → server.
// base may be nil/empty when the client did not send a base snapshot (2-way union).
// Auto-merges non-overlapping adds; emits conflicts for delete-vs-update and note conflicts.
func MergeWatchlistReplace(base, client, server []WatchlistItem) (merged []WatchlistItem, conflicts []WatchlistConflictItem) {
	baseMap := ItemsByKey(base)
	clientMap := ItemsByKey(client)
	serverMap := ItemsByKey(server)

	// Union of all keys that appear in any side.
	keys := map[string]struct{}{}
	for k := range baseMap {
		keys[k] = struct{}{}
	}
	for k := range clientMap {
		keys[k] = struct{}{}
	}
	for k := range serverMap {
		keys[k] = struct{}{}
	}

	for k := range keys {
		_, inBase := baseMap[k]
		cItem, inClient := clientMap[k]
		sItem, inServer := serverMap[k]

		switch {
		case inClient && inServer:
			if cItem.SameContent(sItem) {
				merged = append(merged, sItem)
				continue
			}
			// Both present with different notes → conflict (unless base helps: only one side changed)
			if inBase {
				b := baseMap[k]
				clientChanged := !cItem.SameContent(b)
				serverChanged := !sItem.SameContent(b)
				if clientChanged && !serverChanged {
					merged = append(merged, cItem)
					continue
				}
				if serverChanged && !clientChanged {
					merged = append(merged, sItem)
					continue
				}
			}
			ci, si := cItem, sItem
			conflicts = append(conflicts, WatchlistConflictItem{
				Exchange: sItem.Exchange, Symbol: sItem.Symbol,
				Type: ConflictUpdateVsUpdate, ServerItem: &si, ClientItem: &ci,
			})
		case inClient && !inServer:
			// Client has it, server doesn't.
			if inBase {
				// Server deleted; client kept/changed → conflict
				ci := cItem
				conflicts = append(conflicts, WatchlistConflictItem{
					Exchange: cItem.Exchange, Symbol: cItem.Symbol,
					Type: ConflictUpdateVsDelete, ServerItem: nil, ClientItem: &ci,
				})
			} else {
				// Client added (or no base: treat as client add) → auto-merge
				merged = append(merged, cItem)
			}
		case !inClient && inServer:
			// Server has it, client doesn't.
			if inBase {
				// Client deleted; server kept/changed → conflict if server changed note, else still user choice
				// User asked: deleted on one side and changed on the other → choose.
				// Also deleted vs unchanged on server: still let user confirm? Spec: "deleted on one side and changed on the other"
				// If server unchanged from base, client delete wins (common OT).
				b := baseMap[k]
				if sItem.SameContent(b) {
					// pure delete on client, server unchanged → accept delete
					continue
				}
				si := sItem
				conflicts = append(conflicts, WatchlistConflictItem{
					Exchange: sItem.Exchange, Symbol: sItem.Symbol,
					Type: ConflictDeleteVsUpdate, ServerItem: &si, ClientItem: nil,
				})
			} else {
				// No base: treat as server-only add → keep (other device added)
				merged = append(merged, sItem)
			}
		default:
			// only in base, neither client nor server — already gone
		}
	}
	return merged, conflicts
}

// MergeWatchlistAdd decides whether an add can apply against a newer server version.
// Returns (nil, nil, true, applyItem) when auto-merge should write applyItem.
// Returns (nil, conflict, false, _) when user must resolve.
// Returns (current, nil, true, zero) when no-op success (already present same content).
func MergeWatchlistAdd(server Watchlist, item WatchlistItem) (noop *Watchlist, conflict *WatchlistConflictItem, auto bool, apply WatchlistItem) {
	for _, it := range server.Items {
		if it.Exchange == item.Exchange && it.Symbol == item.Symbol {
			if it.SameContent(item) {
				cp := server
				return &cp, nil, true, WatchlistItem{}
			}
			si, ci := it, item
			return nil, &WatchlistConflictItem{
				Exchange: item.Exchange, Symbol: item.Symbol,
				Type: ConflictUpdateVsUpdate, ServerItem: &si, ClientItem: &ci,
			}, false, WatchlistItem{}
		}
	}
	// Not on server → auto-add
	return nil, nil, true, item
}

// MergeWatchlistRemove decides remove against a newer server version.
func MergeWatchlistRemove(server Watchlist, exchange Exchange, symbol string) (noop *Watchlist, conflict *WatchlistConflictItem, autoRemove bool) {
	for _, it := range server.Items {
		if it.Exchange == exchange && it.Symbol == symbol {
			// Server still has it; client wants delete after concurrent change → conflict
			si := it
			return nil, &WatchlistConflictItem{
				Exchange: exchange, Symbol: symbol,
				Type: ConflictDeleteVsUpdate, ServerItem: &si, ClientItem: nil,
			}, false
		}
	}
	// Already gone → no-op success
	cp := server
	return &cp, nil, false
}
