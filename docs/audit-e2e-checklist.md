# Audit E2E Checklist

## Scope
- Validate audit write coverage for auth, user, role, rbac, plugin config, and system settings.
- Validate query capabilities on audit page (time range + actor + action + resource).
- Validate detail drawer fields include before/after snapshots for update flows.

## Preconditions
1. Start backend with file logging enabled.
2. Ensure frontend points to the same backend instance.
3. Login with admin account.

## Runtime Setup
```powershell
$env:SKOLL_LOG_DIR="log"
$env:SKOLL_LOG_FILE="skoll.log"
go run ./cmd/skoll
```

## One-Command Smoke Check
```powershell
powershell -ExecutionPolicy Bypass -File ./scripts/smoke-auth-audit.ps1
```

Optional parameters:
```powershell
powershell -ExecutionPolicy Bypass -File ./scripts/smoke-auth-audit.ps1 -BasePrefix "http://127.0.0.1:8080/skoll" -Account "admin" -Password "Admin@123456"
```

## Scenario Matrix
1. Auth flow
- Trigger login success, login failed, and logout.
- Expect actions: `login`, `login_failed`, `logout`.

2. User flow
- Create user, update user, update email, disable user, assign role, delete user.
- Expect resources: `user`.
- For update actions, detail contains `before` and `after`.

3. Role flow
- Create role, grant permission, revoke permission, update role, delete role.
- Expect resources: `role`.
- For update action, detail contains `before` and `after`.

4. RBAC flow
- Bind role to user, set role policies, unbind binding.
- Expect resources: `rbac`.

5. System settings flow
- Upsert a key and reset settings.
- Expect resource: `system_setting`.
- Upsert detail contains `before` and `after`.

6. Plugin config flow
- Update plugin config.
- Expect resource: `plugin` and action `update_config`.
- Detail contains `before` and `after` key-count summary.

## UI Verification
1. Open audit page.
- No record is selected by default on first load.

2. Apply filters.
- Use combinations of `actorId + action + resource + from/to`.
- Verify returned rows match all conditions.

3. Open detail drawer.
- Verify detail JSON includes expected metadata and diff fields.

4. Export CSV.
- Verify downloaded file row count matches current query result.

## Log Verification
```powershell
Select-String -Path log/skoll.log -Pattern "append auth audit failed|login_failed|logout|update_config|set_policies|unbind|upsert|reset"
```
- No `append auth audit failed` should appear in normal path.
- Expected action keywords should appear after each scenario.

## Sign-off
- [ ] Backend API checks passed
- [ ] Frontend query and detail checks passed
- [ ] Export checks passed
- [ ] Log checks passed
- [ ] Regression tests and build passed
