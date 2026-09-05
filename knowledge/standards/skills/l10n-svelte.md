---
id: l10n-svelte
name: Localization catalogue - SvelteKit
---

The SvelteKit instantiation of `l10n`: translation resolves on the server before the page renders,
the root layout's `load()` negotiates the locale (a cookie first, then `Accept-Language`), and no
i18n runtime or catalogue copy reaches the client bundle. Reach for it whenever adding or editing any
user-facing string in a SvelteKit project, or wiring locale negotiation in the root `load()`.

The rules below are the composed l10n-svelte rule; it composes alongside `l10n`.
