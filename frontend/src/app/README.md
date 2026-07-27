# app/

Application bootstrap: providers, router, root layout, error boundary.

| Module | Role |
| --- | --- |
| `providers.tsx` | Redux, theme, Ant Design locale, **ErrorBoundary**, router |
| `ErrorBoundary.tsx` | Catch render failures → retry / reload UI |
| `App.tsx` | Shell chrome (nav, language, footer) |
| `routes.tsx` | SPA routes |

- Import Redux store from `@/libs/api` (or `libs/api/store`)
- Keep this layer thin — no business rules
