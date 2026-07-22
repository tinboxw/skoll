# Managed Plugin Process Contract

This document defines the only current runtime for plugins with an independent backend. The backend must be shipped in the plugin package and started by the Skoll lifecycle. Pre-started remote services, arbitrary commands, and alternate entries are not supported.

## Package Entry

The executable path is fixed per platform:

| Platform | Path |
| --- | --- |
| Windows | `backend/bin/<plugin-id>-server.exe` |
| Linux/macOS | `backend/bin/<plugin-id>-server` |

A plugin that declares `service_base_url` must contain the current-platform entry when packaged. The entry must be a regular file inside the installed plugin directory. Missing files, symlinks, directories, and non-executable Unix files prevent enablement.

## Network Boundary

`service_base_url` and `service_health_url` must use plain HTTP on the same `localhost` or loopback-IP origin. User information, query strings, and fragments are rejected. Skoll passes the origin address as `SKOLL_PLUGIN_ADDRESS`; the plugin must listen there and return `2xx` from the declared health path.

## Process Environment

The child receives only the operating-system path, temporary-directory, and user-directory variables needed to run, plus:

- `SKOLL_PLUGIN_ID`: validated Manifest identity;
- `SKOLL_PLUGIN_ADDRESS`: service listen address;
- `SKOLL_PLUGIN_DIR`: installed plugin root.
- `SKOLL_PLUGIN_HOST_URL`: loopback-only host-service v1 endpoint;
- `SKOLL_PLUGIN_HOST_TOKEN`: credential valid only for this managed process lifecycle.

JWT secrets, database credentials, and unrelated `SKOLL_*` values are not inherited. Host capabilities are consumed through `pkg/pluginclient`; the lifecycle token is revoked on failed start, crash, disable, uninstall, or shutdown and is never reused.

## Lifecycle

1. Install validates the Manifest, checksum, and package entry without starting it.
2. Enable runs current migrations, starts the backend, and waits for readiness within the lifecycle budget.
3. Process exit or health failure moves the service to Failed and closes business traffic.
4. Disable closes routes and subscriptions before stopping the process.
5. Uninstall stops the process before applying the one declared data policy.
6. Host shutdown stops every managed plugin process.

Windows terminates the process directly. Unix sends an interrupt first and force-stops after the timeout. Lifecycle audit records stable state codes and never records the child environment.
