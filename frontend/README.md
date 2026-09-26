# Carpool frontend

Next.js App Router frontend for the RiceApps student carpool project. It supports public ride and location browsing once the assigned backend reads are implemented. Posting, joining, ride history, and profile pages require sign-in and onboarding.

Run `npm run dev` from this directory, or `npm run dev` from the repository root to start Supabase, the Go API, and the frontend together. Next.js runs on port 3000 locally.

For a standalone frontend process, set `NEXT_PUBLIC_SUPABASE_URL`, `NEXT_PUBLIC_SUPABASE_ANON_KEY`, and optionally `NEXT_PUBLIC_API_URL` (defaults to `http://localhost:8080`). These values are public browser configuration, not secrets. The root launcher supplies the Supabase values automatically.

`npm run generate` regenerates `src/gen` from `../proto`. Do not edit generated files by hand.

Run `npm test`, `npm run lint`, `npx tsc --noEmit`, and `npm run build` to check frontend changes. For the mocked sign-out privacy and onboarding browser tests, run `npx playwright install chromium` once, then `npm run test:e2e`. These browser tests do not need a running backend or Google credentials.
