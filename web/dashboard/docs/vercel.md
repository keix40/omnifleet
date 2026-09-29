# Vercel deployment (dashboard)

1. Import the repository and set **Root Directory** to `web/dashboard`.
2. Framework preset: **Next.js**.
3. Environment variables (Production + Preview):

| Name | Example |
|------|---------|
| `NEXT_PUBLIC_GATEWAY_URL` | `https://your-api.onrender.com` |
| `NEXT_PUBLIC_WS_URL` | `wss://your-api.onrender.com` |

4. Deploy. The map and platform panels call the Render-hosted `omnifleet-all` gateway.
