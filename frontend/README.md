# Carpool frontend

Next.js App Router frontend for the RiceApps carpool reference app.

Run `npm run dev` from this directory, or `npm run dev` from the repository root to start Supabase, the Go API, and the frontend together. Next.js runs on port 3000 locally.

For a standalone frontend process, set `NEXT_PUBLIC_SUPABASE_URL`, `NEXT_PUBLIC_SUPABASE_ANON_KEY`, and optionally `NEXT_PUBLIC_API_URL` (defaults to `http://localhost:8080`). These values are public browser configuration, not secrets. The root launcher supplies the Supabase values automatically.

`npm run generate` regenerates `src/gen` from `../proto`. Do not edit generated files by hand.
