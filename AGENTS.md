# Agent Instructions

Do not kill/restart dev processes unless asked or you started them.

Dev mode: backend `127.0.0.1:34115`, frontend HMR `127.0.0.1:5173`.
`go run . -start` serves built assets, not HMR.

## Validation Workflow

- Use existing dev services when they are already running. Only start missing
  services yourself, and only stop processes you started.
- API smoke is read-only and targets the backend on `127.0.0.1:34115`:

  ```powershell
  powershell -NoProfile -ExecutionPolicy Bypass -File scripts/smoke-api.ps1 -BaseUrl http://127.0.0.1:34115
  ```

- Browser smoke targets frontend HMR on `127.0.0.1:5173` with the backend on
  `127.0.0.1:34115`. Use hash-router URLs such as `/#/overview/summary`:

  ```powershell
  cd frontend
  npm run test:smoke
  ```

  Override the target with `AGENTMETER_WEB_URL` only when needed.

- Keep routine smoke validation read-only. Do not click **Update Index**,
  **Rebuild Index**, save settings, or apply/change agent privacy settings
  unless the task explicitly requires that state change.
