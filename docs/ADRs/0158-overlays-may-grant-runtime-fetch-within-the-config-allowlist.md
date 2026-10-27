---
title: "158. Overlays may grant runtime fetch within the config-level allowlist"
status: Accepted
relates_to:
  - security-threat-model
topics:
  - harness
  - overlays
  - security
  - remote-resources
---

# 158. Overlays may grant runtime fetch within the config-level allowlist

Date: 2026-10-27

Amends: [ADR 0112](0112-overlays-may-set-any-harness-field.md)

## Status

Accepted

## Context

[ADR 0112](0112-overlays-may-set-any-harness-field.md) Decision 1 lets an
overlay set any harness field except `base`, `trigger`, `overlays`, `slug`,
`role` and `forge`, and says "field merge rules … are unchanged". Three
fields have no overlay merge rule to keep: `allowed_remote_resources`,
`allow_runtime_fetch` and `max_runtime_fetches`
([ADR 0024](0024-harness-definitions.md)) exist only at the top
level of a harness, and `base:` composition
([ADR 0045](0045-forge-portable-harness-schema.md)) drops them so that a base
cannot widen the child's fetch grants. Once ADR 0112 Decision 1 is implemented, an overlay
could set them, and without a decision nothing would apply them. Four path
fields of a harness fetched from a URL, `agent`, `agent_input`, `plugins` and
`api_servers`, are in the same position: Decision 1 opens them to overlays,
and composition has no overlay rule that resolves them against the source
(the second known limitation under Decision).

The runtime fetch service authorizes each fetch against the composed harness's
own `allowed_remote_resources`. [ADR 0038](0038-universal-harness-access.md)
(resolved design question 1) already requires that list to be a subset of the
config-level `allowed_remote_resources`: an entry is covered when, after
lower-casing and path normalization, it starts with a config-level prefix.
ADR 0038's implementation note says harness validation runs that check; in
fact the commands run it, and `run`, `lint` and `lock` run it only when the
harness has URL references (see the known limitation below). An overlay that
could add any prefix would widen what the sandbox may fetch past what the
repository's configuration allows, and fetched content is an injection surface
([security threat
model](../problems/security-threat-model.md#persistent-injection-via-externally-editable-resources)).

This ADR aims to let one harness enable runtime fetch for one context without
widening what the sandbox may fetch past the repository's configuration.

## Options

### Keep the fetch fields out of overlays

This is the simplest rule: a harness that needs fetch only for some runs
composes a second harness through `base:`, and its trigger must split events
with the first so that each event runs one of them. It leaves ADR 0112
Decision 1 short of "any field" for three fields whose grant the configuration
already bounds.

### Let a matching overlay set them, bounded by the config-level allowlist

An overlay turns fetch on for one context, and every prefix it adds must lie
inside the list the repository's configuration allows, so no overlay widens
what the sandbox may fetch. This is the option this ADR takes.

## Decision

We let a matching overlay grant runtime fetch, bounded by the config-level
allowlist, so that fetch can be turned on per context.

A matching overlay applies `allowed_remote_resources` (concatenated after the
top-level list), `allow_runtime_fetch` (turned on) and `max_runtime_fetches`
(replaced, within the 1 to 1000 range ADR 0024's validation enforces).
Within one harness these are the same author's values, unlike a base's. They
are guarded fields, so they apply only together with ADR 0112 Decision 2: the
overlay's `when:` may read only trusted facts. They are overlay fields only; a
`forge:` block cannot set them.

Every prefix an overlay adds to `allowed_remote_resources` must be covered by
the config-level list under ADR 0038's subset test, or the harness fails to
load naming the overlay index and the entry. The check runs when the harness
is composed, which `lint`, `lock`, `run` and dispatch do with the config-level
list, not in harness validation, which has no such list. Where overlay entries
are checked without that list, every prefix an overlay adds is refused. The
check runs on every overlay entry whatever its `when:` says, and whether or not
the harness has URL references.

The config-level list is the one the command resolves from the repository's
configuration layers, including fullsend's default prefixes; a `[]` with no
non-empty list above it refuses every prefix an overlay adds (the commands do
not yet resolve it identically over a base layer's `[]`).

Harness validation's cross-field rules (`allow_runtime_fetch` needs a
non-empty list; `max_runtime_fetches` needs `allow_runtime_fetch`) apply to the
merged harness, so they are checked for an overlay entry only when it matches.
A non-empty merged list also starts the fetch service on its own (the
deprecated implicit opt-in, #7155), so an overlay that only adds a prefix
grants fetch too, still within the bound; when the harness has no URL-directory
resources, the merged harness then draws the deprecation warning that `run`,
`lint` and `lock` print.
An overlay that grants fetch should set `allow_runtime_fetch: true` as well.

A base layer's overlays follow the base rule: they resolve inside the base
before the merge, and the fetch fields they set do not reach the child. Their
entries are still checked against the config-level list when the base loads.

**Known limitation:** this decision bounds overlay entries only. A top-level
`allowed_remote_resources` on a harness with no URL references, or whose only
remote reference is a URL `base:`, is still not checked against the
config-level list, and today only `run`, `lint` and
`lock` check the top-level list; harness dispatch and
`fullsend dispatch --route` do not. Closing that is separate work.

**Known limitation:** composition resolves a URL-fetched harness's paths
against the source, and for overlays it resolves only the fields a `forge:`
block can also set, so it has no overlay resolver for `agent`, `agent_input`,
`plugins` or `api_servers`. `agent_input` is a directory, and composition does
not fetch it from a URL even at the top level (it drops the field), so lifting
the narrowing below for `agent_input` also needs a directory fetch. Until
composition resolves these four fields for overlays, an overlay of a harness
fetched from a URL may not set them. This is an interim narrowing of ADR 0112
Decision 1, and it fails closed: the harness fails to load naming the overlay
and the field. Composition is the one place the refusal runs: every command
that loads a harness fetched from a URL tells composition its source. The refusal is implemented
with Decision 1; until then no overlay can set those fields, so nothing needs
to refuse them. An overlay of a local harness, including one that composes a
URL harness through `base:`, may set them.

## Forge scope

Forge-neutral as a decision: composition and the bound are the same on
GitHub, GitLab and Forgejo. The runtime fetch service itself fetches only
GitHub-hosted URLs today. It refuses a URL by the URL's host, not by the run's
forge, so a grant fetches GitHub-hosted URLs on any forge's run, and a prefix on
another forge's host fetches nothing until fetch support lands there.

## Consequences

- One harness can enable runtime fetch for one context (for example
  `runtime.forge == "github"`) without enabling it everywhere.
- The merge is additive only: an overlay can turn fetch on or add prefixes,
  not turn it off or narrow the list, except the per-run fetch cap, which it
  can lower as well as raise within the range validation allows.
- A harness with an overlay grant outside the config-level list, which had no
  effect before, now fails to load: `run`, `lint` and `lock` fail, and harness
  dispatch skips the agent with an error annotation, so dispatch for that event
  and the Jira poll cycle fail when no registered agent loads.
- Where routing does not load the harness, the stage still starts and then
  fails in `run`: `fullsend dispatch --route` routes it for an actor with write
  access or under an event-level exception (such as a label add or a recognized
  bot) or retro's lifecycle-close exception, and the GitLab poller and webhook
  drivers route by agent name.
