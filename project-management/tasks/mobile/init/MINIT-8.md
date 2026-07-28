# MINIT-8: Home + Markets stub pages with ViewModels

| Field | Value |
|---|---|
| **ID** | MINIT-8 |
| **Epic** | mobile-project-initialization |
| **Status** | done |
| **Area** | mobile |

## Summary

modules/app/pages/HomePage: View + useHomePageViewModel (useGetHealthQuery, AppState pollingInterval, refetchOnFocus false, injectable viewModel prop). modules/markets/pages/MarketsPage: placeholder stub VM only. Wire into navigators. View tests inject VM; VM tests mock RTK. Views must not import RTK.

## Status

Update status in this file and in `project-management/board.md` when work starts/finishes.
