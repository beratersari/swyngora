# libs/hooks

Shared React hooks used by multiple features/pages.

Examples (init / markets epics):

| Hook                 | Purpose                       |
| -------------------- | ----------------------------- |
| `useDocumentVisible` | Pause polling when tab hidden |
| `useDebouncedValue`  | Debounce search `q`           |
| `useMediaQuery`      | Match CSS breakpoints (chart height, etc.) |
| `useDisplayCurrency` | Header FX preference + convert/format helpers |
| `useDeskPriceTape` | Sticky header tape source + live last prices (current venue only; previous RTK list is not relabeled) |

Feature-specific composition hooks may live next to the feature **only if** they are not reusable; prefer promoting shared ones here.

**Do not** put RTK endpoint definitions here — those belong in `libs/api`.
