# libs/i18n (mobile)

Flexible client localization for the React Native / react-native-web app.

## Stack

- **i18next** + **react-i18next**
- **i18next-browser-languagedetector** (localStorage → navigator) on web
- Bundled JSON catalogs (no network fetch for core UI)
- Storage key: `swyngora.mobile.locale.v1` (see `config.ts`)

## Layout

```text
libs/i18n/
├── config.ts           # locales, namespaces, storage key, meta
├── resources.ts        # register locale JSON bundles
├── i18n.ts             # init + setAppLocale / getCurrentLocale
├── useLocale.ts        # hook for switchers
├── types.ts            # typed resources from en catalogs
├── locales/
│   ├── en/{common,home,markets,watchlist,pumps,detail}.json
│   └── tr/...
└── index.ts
```

## Usage

```tsx
import { useTranslation } from 'react-i18next';

const { t } = useTranslation('markets');
return <Text>{t('title')}</Text>;
```

Cross-namespace: `t('common:actions.retry')` or `useTranslation(['markets', 'common'])`.

ViewModels may call `useTranslation` (or `i18n.t` + `i18n.language` deps) so copy updates on language change.

```tsx
import { useLocale } from '@/libs/i18n';

const { locale, options, setLocale } = useLocale();
await setLocale('tr');
```

## Add a language

1. Create `locales/<code>/{common,home,markets,watchlist,pumps,detail}.json`
2. Import + register in `resources.ts` (`localeBundles`)
3. Add `<code>` to `SUPPORTED_LOCALES` and `LOCALE_META` in `config.ts`
4. Add `common.language.<code>` labels in each locale’s `common.json`

## Add a namespace

1. Add name to `I18N_NAMESPACES` in `config.ts`
2. Create JSON for every supported locale
3. Import + attach in `resources.ts`

## Rules

- UI copy lives in locale JSON — not hard-coded long-term
- Prefer short key paths: `nav.markets`, `actions.retry`
- API/domain identifiers (exchange ids, symbols) stay untranslated
- Default language: **en**; also ship **tr**
