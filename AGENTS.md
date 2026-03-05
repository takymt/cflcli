# AGENTS.md

## Purpose

This repository is building a Confluence CLI.

Unless the user says otherwise, Codex should operate autonomously for low-risk specification and documentation work, and ask only when product behavior or external impact is unclear.

## Autonomous Actions

Codex may do the following without asking for confirmation:

- inspect the repository structure and existing files
- create or update product spec documents under `docs/product-specs/`
- translate or rewrite documentation without changing meaning
- refine command names, output wording, and spec phrasing when the underlying behavior is already decided
- add clarifying sections such as assumptions, non-goals, error handling, and examples to spec documents
- choose reasonable defaults for low-risk implementation details when they do not affect product scope

## Ask Before Deciding

Codex must ask the user before deciding any of the following:

- product behavior that changes user-visible CLI semantics
- command scope, feature scope, or MVP boundaries
- API compatibility assumptions that may affect future implementation
- data model choices that become part of the public contract
- destructive or irreversible file operations

## Defaults For This Repo

When the user has not specified otherwise, use these defaults:

- write product specs in English
- place product specs in `docs/product-specs/`
- keep specs concise and implementation-oriented
- prefer `cfl page ...` command grouping for page features
- treat the local Markdown file as the source of truth for sync behavior

## Editing Guidance

- prefer minimal edits over broad rewrites
- preserve previously approved behavior
- do not silently change product decisions already confirmed by the user
- after completing requested code or script changes, commit them before asking for review or feedback
- when the user asks for changes to be committed incrementally, commit the current change before asking for feedback or proposing the next step
- do not run state-changing git commands such as `git add` and `git commit` in parallel
