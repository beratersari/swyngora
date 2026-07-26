# Epic C: Mobile project initialization

**Priority:** P0 (mobile track)  
**Status:** done  
**Blocks:** multi-exchange spot markets (mobile)  
**Design plan:** `docs/design/mobile-project-initialization.md`  
**System design:** `docs/design/mobile-system-design.md`  
**Decision:** `project-management/decisions/002-react-native-cli-modules-viewmodel.md`

## Goal

Scaffold `mobile/` with **React Native CLI** (no Expo), TypeScript, Atomic Design (**no pages in components**), **modules that own pages**, **View + ViewModel** per page, RTK Query, OpenAPI codegen, AppState hygiene, and **the same brand color tokens as `frontend/`**.

## Tasks

- [x] MINIT-1 Scaffold React Native CLI TypeScript (no Expo)
- [x] MINIT-2 Lint, format, Jest, path aliases
- [x] MINIT-3 libs + Atomic + modules skeleton + boundary ESLint
- [x] MINIT-4 libs/api store + baseApi + env (absolute URL)
- [x] MINIT-5 OpenAPI codegen
- [x] MINIT-6 Navigation shell + AppState + providers
- [x] MINIT-7 Color tokens (match frontend) + core atoms + ScreenTemplate
- [x] MINIT-8 Home + Markets stub pages with ViewModels
- [x] MINIT-9 Package docs + root AGENTS/README + changelog

## Acceptance

- `npm run android` launches shell (documented host)
- `npm run codegen:api` works
- Brand hex in `mobile/src/styles/tokens/colors.ts` matches frontend
- No Expo deps; no `components/pages/`
- Home health ViewModel works against local backend when running
- Structure matches `mobile/AGENTS.md`
