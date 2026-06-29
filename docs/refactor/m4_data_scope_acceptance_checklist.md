# M4 Data Scope Acceptance Checklist

> Work Item: M4-07-02  
> Scope: admin, department manager, normal user data-scope verification for organization and user list workflows.  
> Canonical scopes: `all`, `department`, `department_tree`, `self`, `custom`.

## Preconditions

1. Backend and frontend are running against the same data store.
2. The organization catalog contains at least:
   - `dept-root`
   - `dept-sales`
   - `dept-sales-east`, child of `dept-sales`
   - `dept-ops`
3. The user list contains at least:
   - `admin_user`, department `dept-root`
   - `sales_manager`, department `dept-sales`
   - `sales_east_user`, department `dept-sales-east`
   - `ops_user`, department `dept-ops`
4. Roles and policy bindings are prepared:
   - `super_admin`: role key `super_admin`, expected bypass/all data.
   - `department_manager`: data scope `department_tree`, actor department `dept-sales`.
   - `normal_user`: data scope `self`, actor user `sales_east_user`.
   - Optional `custom_scope_user`: data scope `custom`, custom departments `dept-sales-east` and `dept-ops`.

## Frontend Routes

| Route | Purpose | Expected permission |
|---|---|---|
| `/skoll/user` | user list filtered by backend data scope and local organization filters | `user.read` |
| `/skoll/organization` | department tree, positions, user assignment editing | `org.read`; edit requires `org.manage` and `user.update` |
| `/skoll/permission` | policy rule scope review | `permission.manage` |
| `/skoll/role` | role binding review | `role.read` / `role.manage` |

## Scenario Matrix

| Scenario | Login role | Scope context | User list expectation | Organization page expectation | Result |
|---|---|---|---|---|---|
| Admin bypass | `super_admin` | bypass/all | sees all prepared users | can inspect all departments and edit assignments when granted `org.manage`/`user.update` | Pending |
| Department manager | `department_manager` | `department_tree`, `dept-sales` | sees `sales_manager` and `sales_east_user`; does not see `ops_user` | department filter can isolate `dept-sales` and `dept-sales-east` users | Pending |
| Normal user | `normal_user` | `self`, `sales_east_user` | sees only `sales_east_user` | no organization edit controls unless explicitly granted | Pending |
| Custom scope | `custom_scope_user` | `custom`, `dept-sales-east`,`dept-ops` | sees `sales_east_user` and `ops_user`; does not see `sales_manager` | selected department counts match visible users | Pending |

## Manual Smoke Steps

### 1. Admin

1. Login as `super_admin`.
2. Open `/skoll/user`.
3. Confirm all prepared users are visible.
4. Open `/skoll/organization`.
5. Confirm the department tree displays root, sales, sales-east, and ops.
6. Open a user assignment drawer and save a department/position change.

Expected result: full visibility and successful assignment save.

### 2. Department Manager

1. Login as `department_manager`.
2. Open `/skoll/user`.
3. Confirm only `dept-sales` and `dept-sales-east` users are visible.
4. Open `/skoll/organization`.
5. Select `dept-sales`, then `dept-sales-east`.
6. Confirm user assignment list counts match the selected department.

Expected result: no users outside the manager's data scope are visible.

### 3. Normal User

1. Login as `normal_user`.
2. Open `/skoll/user`.
3. Confirm only the current actor user is visible.
4. Open `/skoll/organization`.
5. Confirm edit controls are disabled or hidden without `org.manage` / `user.update`.

Expected result: self-only list and restricted organization editing.

## API Cross-Checks

Run these calls with each role's token and record the user IDs returned:

```powershell
Invoke-RestMethod -Headers @{ Authorization = "Bearer <token>" } -Uri "http://127.0.0.1:8080/v1/users?offset=0&limit=50"
```

Expected API results must match the UI table for the same role.

## Failure Record

For every failed row in the scenario matrix, record:

| Field | Value |
|---|---|
| Scenario |  |
| Actor role |  |
| Expected users |  |
| Actual users |  |
| Route/API |  |
| Screenshot or command output |  |
| Fix task |  |
| Retest result |  |

## Acceptance Gate

- The scenario matrix has no failed rows.
- UI and API results match for admin, department manager, and normal user.
- The organization page records default, loading, empty, backend-error, no-permission, save-success/failure, and narrow viewport coverage in `acceptance_log.md`.
- `cd web && npm run build` passes.
