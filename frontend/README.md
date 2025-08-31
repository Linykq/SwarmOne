# SwarmOne Frontend (a0.dev-inspired)

Vite + React + TS + Tailwind. Glass cards, gradient grid background, pill nav.
Home page: Task builder (Task/Content/Expections/Source/Language) + live instruction/JSON preview + Answer panel.

## Dev
```bash
npm i
npm run dev
# http://localhost:5173
```
Dev proxy sends `/v1/*` and `/health` to `http://localhost:8080`.

## Prod
Set `VITE_API_BASE` if backend is on a different origin.
