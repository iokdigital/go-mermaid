# Claude Code Agents Guide (go-mermaid)

## Delivery Workflow

The universal rule: **all work lands on `develop` first; `main` only ever receives `develop` via PR.**

1. **Worktree** — create an isolated worktree branched from `develop`:
   ```bash
   git worktree add .worktrees/<name> -b <type>/description origin/develop
   ```
   If `develop` does not exist yet, create it from `main` first (`git branch develop main && git push -u origin develop`), then branch from it.
2. **PR to develop** — push the worktree branch, open a PR with base `develop`. Never target `main` with feature work. CI must be green.
3. **Test the develop deployment** — verify the change on the deployment built from `develop` before release.
4. **PR to main** — after acceptance, open a PR from `develop` to `main`.
5. **Tag on merge** — when the `develop`→`main` PR merges, tag the release per repo convention so production deploys the tagged build. Tags live on `main` only.

**Never:** commit directly to `main`; open feature PRs against `main`; deploy production from `develop`; tag `develop` commits.
