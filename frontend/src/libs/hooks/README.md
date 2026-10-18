# libs/hooks

Shared React hooks used by multiple features/pages.

Examples (init / markets epics):

| Hook                 | Purpose                                                                                                                                                                                                                |
| -------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `useDocumentVisible` | Pause polling when tab hidden                                                                                                                                                                                          |
| `useDebouncedValue`  | Debounce search `q`                                                                                                                                                                                                    |
| `useOrderPreview`    | Debounced paper order cost and risk. A risk percent plus a stop asks the preview for the size. The page ticket host owns the query so a keystroke does not render the rest of the desk. The form stays presentational. |
| `useExecutionCompare` | Debounced fee and slippage for one crypto size on this venue, as best cost, and as a time split. Keeps the last success and its clock while a newer read is in flight. The ticket host owns the query. |
| `useMediaQuery`      | Match CSS breakpoints (chart height, etc.)                                                                                                                                                                             |
| `useDisplayCurrency` | Header FX preference + convert/format helpers                                                                                                                                                                          |
| `useDeskPriceTape`   | Sticky header tape source + live last prices (current venue only; previous RTK list is not relabeled)                                                                                                                  |

Feature-specific composition hooks may live next to the feature **only if** they are not reusable; prefer promoting shared ones here.

**Do not** put RTK endpoint definitions here — those belong in `libs/api`.
