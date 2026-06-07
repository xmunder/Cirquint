# Circuit Review Workbench Web

Minimal Next.js scaffold for the review workbench MVP.

## Environment

Set `BACKEND_BASE_URL` to the Go API origin used by the review routes.

Example:

```bash
BACKEND_BASE_URL=http://127.0.0.1:8080
```

The typed review client reads this variable on the server and targets:

- `GET /projects/{projectId}/circuit-reviews`
- `GET /projects/{projectId}/circuit-reviews/{jobId}`
- `POST /projects/{projectId}/circuit-reviews/{jobId}/decision`

## Local verification

```bash
npm install
npm test
npx tsc --noEmit
```
