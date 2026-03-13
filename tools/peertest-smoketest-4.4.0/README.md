# Peer Test Smoke Test

Standalone Playwright smoke test for a local WSO2 APIM peer test pack.

## What it does

1. Logs into Carbon as the super tenant admin.
2. Ensures tenant `peertest.com` exists.
3. Logs into Carbon as the tenant admin `peer@peertest.com`.
4. Ensures tenant user `peertestuser` exists with roles:
   - `Internal/creator`
   - `Internal/publisher`
   - `Internal/subscriber`
5. Logs into Publisher as `peertestuser@peertest.com`.
6. Creates and publishes a REST API against `https://httpbin.org/anything`.
7. Opens the API in Dev Portal from the Publisher overview.
8. Subscribes `DefaultApplication` to the created API.
9. Generates production keys for `DefaultApplication`.
10. Opens the API Console, gets a test key, and executes the `GET /*` operation.

The script is idempotent enough for local use:
- tenant creation is skipped if the tenant already exists
- user creation is skipped if the user already exists
- API name is timestamped to avoid collisions

## Install

```bash
cd /Users/tharsanan/Documents/ai-helper/tools/peertest-smoketest-4.4.0
npm install
npx playwright install chromium
```

## Run headed

```bash
cd /Users/tharsanan/Documents/ai-helper/tools/peertest-smoketest-4.4.0
npm run smoketest -- \
  --base-url https://localhost:9443 \
  --admin-user admin \
  --admin-password admin \
  --tenant-domain peertest.com \
  --tenant-admin-user peer \
  --tenant-admin-password peer1 \
  --tenant-admin-email peer@peertest.com \
  --tenant-user peertestuser \
  --tenant-user-password peer1
```

## Headless

```bash
npm run smoketest -- --headless
```

## Save screenshots

```bash
npm run smoketest -- \
  --screenshot-dir /Users/tharsanan/Documents/tmp/peertest-smoketest-shots \
  --screenshot-delay-ms 1000
```

Screenshots are saved in ordered filenames like:

```text
001-before-submit-carbon-login-admin.png
002-after-carbon-login-admin.png
003-tenant-list-peertest-com.png
...
```

For multi-input forms, the script captures the page just before submit:
- Carbon login
- create tenant
- tenant user step 1
- tenant user role assignment
- Publisher login
- API create form

It also captures the important Dev Portal transitions:
- before opening Dev Portal from Publisher
- subscription dialog before clicking `Subscribe`
- production keys page before clicking `Generate Keys`
- API Console before `GET TEST KEY`
- API Console just before `Execute`
- API Console after the `200` response appears

When screenshots are enabled, the script waits `1000ms` before each capture by default. Adjust that with `--screenshot-delay-ms`.

## Useful flags

- `--base-url https://localhost:9443`
- `--headless`
- `--slow-mo 300`
- `--screenshot-dir /path/to/screenshots`
- `--screenshot-delay-ms 1000`
- `--api-endpoint https://httpbin.org/anything`
- `--api-name-prefix PeerTestAPI`
- `--keep-open`

## Notes

- The script ignores the self-signed local HTTPS certificate.
- It uses direct Carbon URLs for stability instead of only left-nav clicks.
- It assumes the tenant admin login is `peer@peertest.com` style.
- It uses `DefaultApplication` for subscriptions and key generation.
