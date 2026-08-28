# Phase 1. Devin argv

Back to [overview](overview.md).

## Goal

`buildDevinArgs` emits the 3000.6.2 `--print` contract. Prompt text is not on argv. Custom args cannot break print mode, permissions, or resume.

## Changes

- New `server/pkg/agent/devin.go` with `devinBlockedArgs` and `buildDevinArgs`.
- Tests in `devin_test.go` for fresh argv, resume argv, blocked flags, and ExtraArgs order.

Do not spawn a process yet.

## Data structures

`devinBlockedArgs` owns `-p`, `--print`, `--prompt-file`, `--permission-mode`, `--respect-workspace-trust`, `-c`, `--continue`, `-r`, `--resume`, `--sandbox`, `acp`, `cloud`, `ssh`, and `desktop`.

## Verification

**Static.** `cd server && go test ./pkg/agent -run 'TestBuildDevin|TestFilterDevin' -count=1`

**Runtime.** None. Argv only.
