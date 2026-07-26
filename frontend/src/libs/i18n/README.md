# libs/i18n

Flexible client localization for the product web app.

## Stack

- **i18next** + **react-i18next**
- **i18next-browser-languagedetector** (localStorage → navigator)
- Bundled JSON catalogs (no network fetch required for core UI)

## Layout

```text
libs/i18n/
├── config.ts           # locales, namespaces, storage key
├── resources.ts        # register locale JSON
├── i18n.ts             # init
├── antdLocale.ts       # Ant Design locale packs
├── types.ts            # typed keys from en resources
├── locales/
│   ├── en/{common,markets,detail}.json
│   └── tr/{common,markets,detail}.json
└── index.ts
```

## Usage

```tsx
import { useTranslation } from 'react-i18next';

const { t } = useTranslation('markets');
return <span>{t('title')}</span>;
```

Cross-namespace: `t('common:actions.retry')` or `useTranslation(['markets', 'common'])`.

## Add a language

1. Create `locales/<code>/{common,markets,detail}.json`
2. Register in `resources.ts`
3. Add `<code>` to `SUPPORTED_LOCALES` in `config.ts`
4. Map Ant Design locale in `antdLocale.ts` if available

## Rules

- UI copy lives in locale JSON — not hard-coded in components
- Prefer short key paths: `table.symbol`, `errors.network`
- API/domain identifiers (exchange ids, symbols) stay untranslated
