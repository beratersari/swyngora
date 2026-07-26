# components (Atomic Design)

**Atoms → molecules → organisms → templates only.**

**Do not put pages here.** Pages live under `src/modules/<name>/pages/`.

Lower levels must not import higher levels. Components must not import `modules/` or `app/`.
