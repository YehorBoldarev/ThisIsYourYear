# Step B2 — API client & auth flow

> **Status:** Draft for approval
> **Parent:** [`00-system-plan.md`](00-system-plan.md) §6 (step B2)
> **Builds on:** B1 (scaffold, tokens, primitives, shell) — **done & approved**.
> **Approval gate:** No code is written until you approve this document.

---

## 1. Objective

Give the app a **typed API client** generated from the checked-in
[`../../api/openapi.yaml`](../../api/openapi.yaml), and a real **authentication
flow**: register, log in, **stay logged in across reloads** (silent refresh via
the HttpOnly cookie), reach protected routes, see your real identity in the
header, and log out. After B2 the app talks to the actual Go API.

### In scope
- **Typed client**: `openapi-typescript` generates types from the OpenAPI spec;
  a thin `openapi-fetch` wrapper adds the access-token header and **refresh-on-401
  retry** (single-flight).
- **Auth model**: access token held **in memory**; on boot the app attempts a
  **silent refresh** (the cookie survives reloads); a `401` on any call triggers
  one refresh + retry, then logout if that fails.
- **AuthProvider + `useAuth`**: owns `{ status, user, accessToken }` and
  `login/register/logout`; drives the header identity.
- **Protected routes**: `RequireAuth` gate + redirect to `/login` (preserving the
  intended destination), with a loading state during the boot refresh.
- **Login / Register screens**: on-brand (designed from B1 tokens — these aren't
  in the handoff), with field-level error mapping from the API error envelope and
  a "remember me" option.
- **Header wiring**: replace B1's mock user with the authenticated user; wire the
  theme-toggle-adjacent user block + a logout affordance.
- **TanStack Query** provider installed (used for data from B3; auth itself stays
  in context).
- **Tests**: MSW-backed unit/component tests for the client (token inject,
  401→refresh→retry single-flight), the auth forms (validation + error mapping),
  and the protected-route redirect. Plus a **live** end-to-end check against the
  real Go API + Mongo.

### Explicitly NOT in this step
- No dashboard/topic/goal data screens (B3+). No data-query hooks beyond what auth
  needs. No password reset / email verification (backend has none). The
  Reflections placeholder and gallery stay as-is.

---

## 2. Backend contract (verified against the code)

| Endpoint | Method | Auth | Success | Body in → out |
| --- | --- | --- | --- | --- |
| `/api/v1/auth/register` | POST | none | **201** | `{email, displayName, password}` → `UserResponse` (no token) |
| `/api/v1/auth/login` | POST | none | **200** | `{email, password, rememberMe}` → `TokenResponse` + sets refresh cookie |
| `/api/v1/auth/refresh` | POST | cookie | **200** | — (cookie) → `TokenResponse` (rotates cookie) |
| `/api/v1/auth/logout` | POST | cookie | **204** | — → — (clears cookie) |
| `/api/v1/me` | GET | Bearer | **200** | — → `UserResponse` |

- `TokenResponse = { accessToken, expiresAt, user: UserResponse }`.
  `UserResponse = { id, email, displayName, createdAt }`.
- The **refresh token is an HttpOnly cookie** scoped to path `/api/v1/auth`; JS
  never reads it. In dev the Vite proxy makes the browser see one origin, so the
  cookie flows to `/api/v1/auth/refresh` and `/logout` automatically.
- **Register returns no token** → after a successful register we immediately call
  login with the same credentials (see §7). *Decision: auto-login after register*
  (smoother than bouncing to the login screen; flag if you'd prefer otherwise).
- Errors use the stable envelope `{"error":{"code","message","fields?}}` — e.g.
  `422` validation returns `fields` keyed by field name, which we surface inline.

---

## 3. Small technical decisions (mine; listed for transparency)

| Decision | Choice | Why |
| --- | --- | --- |
| Type generation | **`openapi-typescript`** → `src/api/schema.d.ts` (checked in) | Types straight from the contract; a backend change surfaces as a TS error. A `npm run gen:api` script regenerates it. |
| HTTP wrapper | **`openapi-fetch`** + middleware | Typed paths/bodies, tiny, close to `fetch`; middleware injects the token and handles 401-refresh. |
| Access-token storage | **In memory only** (module holder + context) | Not in `localStorage` (XSS-exfiltration risk). The HttpOnly cookie is the durable credential; memory token is rebuilt by silent refresh. |
| Session restore | **Silent refresh on boot** | Reload loses the memory token; one `POST /refresh` restores it from the cookie. This is what "stay logged in" means here. |
| Refresh concurrency | **Single-flight** shared promise | Many parallel 401s trigger exactly one refresh; all await it. |
| Server state | **TanStack Query** provider added now | Data hooks land in B3; installing the provider here keeps that additive. |
| Form state | Local `useState` + a small validation helper | Two small forms; no form library needed. |
| Test API mocking | **MSW** | Same "verified" bar; deterministic client/flow tests without the real server. |

Auth screen **visuals** aren't in the handoff; per the system plan's open question I
design minimal on-brand screens from the B1 tokens. Flag if you'd rather supply designs.

---

## 4. Files (created ·/ changed)

```
web/
  package.json                    ~ add openapi-fetch, @tanstack/react-query (deps);
                                    openapi-typescript, msw (dev); scripts: gen:api
  src/
    api/
      schema.d.ts                 + generated OpenAPI types (checked in)
      client.ts                   + openapi-fetch client + auth middleware (token, 401-retry)
      tokenStore.ts               + in-memory access-token holder (get/set/clear)
      errors.ts                   + parse the {error:{code,message,fields}} envelope
    auth/
      AuthContext.ts              + context type + createContext
      AuthProvider.tsx            + status/user/token state, boot refresh, login/register/logout
      useAuth.ts                  + hook
      RequireAuth.tsx             + protected-route gate + redirect (saves intended path)
    routes/
      Login.tsx                   + on-brand login screen
      Register.tsx                + on-brand register screen
    components/
      Field.tsx                   + Label+Input+inline error trio (reused by both forms)
      Spinner.tsx                 + small loading indicator (boot + submit states)
    layout/
      Header.tsx                  ~ use real user; add logout control
      AppShell.tsx                ~ take user from useAuth (drop mockUser)
      UserBlock.tsx               ~ accept the auth user shape
    lib/
      mockData.ts                 ~ drop mockUser (mockChapters stay until B3)
    main.tsx                      ~ add QueryClientProvider + AuthProvider
    App.tsx                       ~ split public (/login,/register) vs RequireAuth+AppShell routes
    test/
      msw.ts                      + MSW server + auth handlers
      setup.ts                    ~ start/stop MSW
```

---

## 5. Typed client & 401-refresh

- `gen:api` runs `openapi-typescript ../api/openapi.yaml -o src/api/schema.d.ts`.
  The result is committed so `npm ci` builds don't need the backend present.
- `client.ts` creates `createClient<paths>({ baseUrl: '/' })` and attaches
  middleware:
  - **onRequest**: if a token is in `tokenStore`, set `Authorization: Bearer …`.
    (Requests to `/api/v1/auth/*` skip the header — they use the cookie.)
  - **onResponse**: on `401` for a protected call, call the **single-flight
    `refreshAccessToken()`**; if it resolves, replay the original request once with
    the new token; if it rejects, clear the token and let the failure propagate
    (AuthProvider flips to unauthenticated → redirect).
- `refreshAccessToken()` lives beside the client: it POSTs `/api/v1/auth/refresh`,
  stores the new token, and returns it — guarded so concurrent callers share one
  in-flight promise.

## 6. Auth model & AuthProvider

State machine: `status ∈ { 'loading', 'authenticated', 'unauthenticated' }`.

- **Boot**: `status='loading'`; call `refreshAccessToken()` once.
  Success → set token+user (from the `TokenResponse`), `authenticated`.
  Failure (no/expired cookie → 401) → `unauthenticated`. Either way the loading
  gate resolves.
- **login(email, password, rememberMe)**: POST `/login`; store token+user;
  `authenticated`.
- **register(email, displayName, password)**: POST `/register` (201) → then
  `login(...)` with the same credentials.
- **logout()**: POST `/logout` (best-effort); clear token+user; `unauthenticated`.
- `useAuth()` exposes `{ status, user, login, register, logout }`. The header reads
  `user`; `RequireAuth` reads `status`.

**Why reloads stay logged in:** the access token is in memory (gone on reload), but
the refresh cookie persists; the boot refresh mints a fresh access token from it.

## 7. Screens, routing & header

- **App structure**: `/login` and `/register` render **bare** (no shell). All other
  routes are wrapped by `RequireAuth` → `AppShell` → page. While `status==='loading'`
  the gate shows a centered `Spinner` (no flash of the login screen).
- **RequireAuth**: if `unauthenticated`, `<Navigate to="/login" state={{from}}/>`;
  after login we return the user to `from` (or `/`).
- **Login**: email + password + "Remember me" `Toggle`; submit → `login`. Envelope
  `fields`/`message` map to inline `Field` errors / a form-level error (e.g. `401`
  invalid credentials → "Email or password is incorrect"). Link to Register.
- **Register**: email + display name + password (min 8, surfaced) → `register`
  (auto-login) → redirect to `/`. Duplicate email (`409`) maps to the email field.
- **Header/UserBlock**: show the real `user.displayName` + a role/eyebrow and initials
  derived from the name; add a **logout** control (menu or button). `AppShell` drops
  `mockUser` and reads `useAuth().user`.

## 8. Testing

- **Client** (MSW): token is attached; a `401` triggers exactly one refresh and a
  successful retry; a failing refresh clears the token and surfaces the error;
  concurrent 401s share one refresh (single-flight).
- **AuthProvider**: boot refresh success → `authenticated` with user; boot 401 →
  `unauthenticated`; `register` performs register-then-login; `logout` clears state.
- **Forms** (RTL): required-field validation; API `422 fields` render inline; `401`
  login shows the form-level message; happy path calls the right endpoint.
- **RequireAuth**: unauthenticated renders a redirect to `/login`; authenticated
  renders children.
- **Live** (real API + Mongo): `make compose-up && make run`, then `npm run dev`:
  register a new user, land authenticated, **reload → still authenticated**, see the
  name in the header, log out → bounced to `/login`, and a protected route while
  logged out redirects. Verified in both themes.

## 9. Verification (definition of done)

`gen:api` reproduces `schema.d.ts` with no diff · `typecheck` · `lint` · `format:check`
· `test` (new suites green) · `build`. The live flow in §8 passes against the real
backend in both themes. `web/README.md` (auth/gen notes) and `docs/dev_run.md`
(frontend section) updated. Nothing committed.

## 10. Open questions (non-blocking; defaults stated, proceed unless you object)

- **Auto-login after register** — default **yes** (§2).
- **Auth screen visuals** — I design them on-brand from tokens (system-plan open
  question); supply designs if you'd prefer.
- **User "role" line** — the header shows a role ("Dreamer-in-Chief" in the mock);
  the API has no role field. Default: show a **fixed on-brand eyebrow** (e.g.
  "Dreamer-in-Chief") until/unless a real field exists. Flag to drop it instead.
- **Logout affordance** — default: a small control in the user block (icon/menu);
  refine in B6 polish.

On approval I implement B2, verify per §9 (including the live run), update docs, and
report back for your **"Task completed"**.
