# M1 Frontend Smoke

Date: 2026-06-19

## Automated Check

Run from `web/`:

```powershell
npm run typecheck
npm run build
npm run smoke:m1-permissions
```

The smoke script verifies these source-level invariants:

- Sidebar menus are loaded through the registry-backed navigation store.
- Route guards use the unified route access entry.
- Plugin routes merge permission metadata through the same route access helper.
- Button permission checks use `useButtonAccess` or `v-permission`.
- Menu and permission high-risk changes call the shared confirmation helper.

## Manual Smoke Steps

1. Sign in with a role that has `menu.read`, `plugin.read`, and `user.read`.
2. Confirm the sidebar renders registry menu entries and hides entries whose required permissions are missing.
3. Navigate directly to a route that requires a missing permission and confirm the guard redirects to a safe fallback.
4. Open plugin management and confirm read-only users cannot see or trigger install, enable, disable, uninstall, or release-review actions.
5. Open menu management, edit visibility/order/permission fields, click save, and confirm a warning dialog appears before the request is sent.
6. Open permission management, grant and revoke one permission in the matrix, and confirm each action asks for confirmation and updates the selected role after success.

## Expected Result

The sidebar, route guard, and button permissions agree for the same role and permission set. High-risk menu and permission saves require confirmation and still show page-level error feedback if the API fails.
