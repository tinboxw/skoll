# E14-step3 Upgrade Transaction Checkpoints And Provenance

## Milestone

- ID: `E14-step3`
- Name: `upgrade transaction checkpoints and provenance retention`

## Delivered

- Added checkpointed upgrade transaction execution in plugin governance service:
  - input validation checkpoint
  - current-version load checkpoint
  - signature verification checkpoint
  - dependency precheck checkpoint
  - apply-upgrade checkpoint
  - rollback checkpoint on failed apply
- Added provenance retention for each transaction outcome:
  - transaction id
  - plugin name and version transition
  - success/rollback/reason
  - recorded timestamp
- Added admin APIs:
  - `POST /admin/v1/plugins/{name}/upgrade/transaction`
  - `GET /admin/v1/plugins/upgrade/provenance?limit=20`

## Files

- `internal/module/pluginmgr/service.go`
- `internal/module/pluginmgr/service_test.go`
- `internal/app/admin_modules.go`
- `internal/app/admin_modules_test.go`
- `internal/module/storageadapter/adapter.go`
- `docs/planning/FEATURE_PARITY_PLAN.md`
- `docs/planning/IMPLEMENTATION_ROADMAP.md`
- `README.md`
- `README.en.md`

## Validation

- `go fmt ./...`
- `go test ./internal/module/pluginmgr ./internal/app ./internal/module/storageadapter`
- `go test ./...`
- `go test -race ./...`
