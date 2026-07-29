# MCAT-3: Categories browse page (ViewModel + View + route)

| Field | Value |
|---|---|
| **ID** | MCAT-3 |
| **Epic** | mobile-category-discovery |
| **Status** | done |
| **Area** | mobile |
| **Path** | `project-management/tasks/mobile/category-discovery/MCAT-3.md` |

## Summary

New page under markets module:

```text
mobile/src/modules/markets/pages/categories-page/
  CategoriesPage.tsx
  CategoriesPage.viewModel.ts
  CategoriesPage.types.ts
  CategoriesPage.styles.ts
  CategoriesPage.test.tsx
  index.ts
```

- ViewModel: `useListProductTagsQuery` (exchange = catalog default), search state, featured via MCAT-1 helpers, error/retry  
- View: ScreenTemplate + category grid; no business logic in View  
- Navigation: register `Categories` on **Markets** stack (`navigation.ts` + `RootNavigator` types)  
- Selecting a tag is wired in MCAT-4 (this task may stub `onSelectTag` or call a shared apply helper)

## Design

`docs/design/mobile-category-discovery.md` §4–7

## Acceptance

- [x] Route reachable from Markets stack  
- [x] View + ViewModel split; types in `.types.ts`  
- [x] Tags load, search filters list  
- [x] Featured section only shows live tags  
- [x] Page test covers loading / empty / render tags (mocked query)  
- [x] Status → done when finished  

## Status

`done`
