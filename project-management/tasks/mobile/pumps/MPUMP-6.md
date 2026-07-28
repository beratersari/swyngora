# MPUMP-6: Loading empty error disclaimer and AppState hygiene

| Field | Value |
|---|---|
| **ID** | MPUMP-6 |
| **Epic** | mobile-pumps |
| **Status** | done |
| **Area** | mobile |

## Summary

- Skeleton while scan loading  
- Empty copy with threshold guidance  
- Error + Retry  
- Always show informational disclaimer (`note` or product string)  
- Pause/cancel unnecessary work when app backgrounded; **no scan polling**  
- Debounce filter changes ~300ms  

## Design

`docs/design/mobile-pumps.md` §9–10

## Acceptance

- [ ] Empty/error/loading covered in page tests  
- [ ] Status updated  

## Status

`done`
