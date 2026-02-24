# Task 029: Rename Repo to gh-ghent

- **Phase:** 3.6
- **Status:** DONE
- **Depends on:** None
- **Blocks:** None
- **L4 Visual:** Not required

## Problem

The gh extension research doc states: "All extension repositories MUST be named with the `gh-` prefix." The current repo name `indrasvat/ghent` violates this convention. We worked around it with a manual symlink, but the real fix is renaming the repo.

## Changes

1. Rename GitHub repo from `indrasvat/ghent` to `indrasvat/gh-ghent`
2. Update Go module path: `github.com/indrasvat/ghent` → `github.com/indrasvat/gh-ghent`
3. Update all import paths across 27 .go files
4. Update Makefile, .golangci.yml, docs, and test scripts
5. Update git remote URL
6. Verify with `make ci` and L3 smoke tests

## Acceptance Criteria

- [ ] `go.mod` module is `github.com/indrasvat/gh-ghent`
- [ ] All `.go` imports use `github.com/indrasvat/gh-ghent/...`
- [ ] `make ci` passes (lint + test + vet)
- [ ] `gh extension install .` works natively (no manual symlink)
- [ ] L3 smoke tests pass (`gh ghent checks`, `gh ghent comments`)
