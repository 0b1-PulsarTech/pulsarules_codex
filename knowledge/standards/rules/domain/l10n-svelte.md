---
id: l10n-svelte
name: Localization catalogue - SvelteKit server resolution
description: The SvelteKit instantiation of the neutral localization-catalogue rule - the root layout's load() negotiates the locale server-side before the template renders; no i18n runtime or catalogue copy ships in the client bundle.
tags:
    - svelte
    - i18n
---

# Localization catalogue - SvelteKit server resolution

> In a SvelteKit project, translation resolves on the server before the page renders: the root
> layout's `load()` negotiates the locale (a cookie first, then the `Accept-Language` header) and
> hands the template already-resolved strings. No i18n runtime, and no copy of the catalogue itself,
> ships in the client bundle. This is the SvelteKit instantiation of [[l10n]]'s neutral catalogue
> contract; run both together.

Applies to: a SvelteKit project's UI, which is a template over server-resolved data. Composes with
[[l10n]].

{{define "when"}}
- Adding or editing any user-facing string in a SvelteKit project.
- Wiring or reviewing locale negotiation in the root layout's `load()`.
- Adding a new supported locale to a SvelteKit project.
{{end}}

{{define "must"}}
1. Negotiate the locale on the server, in the root layout's `load()`: a cookie first, then the
   `Accept-Language` header.
2. Hand the template already-resolved strings from `load()`; the template performs no translation
   lookup of its own.
3. Ship no i18n runtime, and no copy of the message catalogue itself, in the client bundle.
{{end}}

{{define "forbidden"}}
- Translating in the client instead of the server-side root `load()`.
- An i18n runtime, translation library, or the message catalogue itself present in the client
  bundle.
{{end}}

{{define "validation"}}
- [ ] Locale negotiation happens in the root layout's `load()`, server-side.
- [ ] The template receives already-resolved strings; it does not translate itself.
- [ ] No i18n runtime or catalogue copy ships to the client bundle.
{{end}}
