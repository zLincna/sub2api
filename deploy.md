# Sub2API Deployment Runbook

## Current Production Topology

- Public URL: `https://ai.lixnus.cc/`
- Target server: `root@103.189.141.223:22`
- Runtime path on server: `/opt/sub2api-deploy`
- Artifact path on server: `/opt/sub2api-artifact`
- Runtime: Docker, local image `sub2api-local:*`
- Database/cache containers:
  - `sub2api-postgres`
  - `sub2api-redis`
- App containers:
  - `sub2api_a` -> `127.0.0.1:8088`
  - `sub2api_b` -> `127.0.0.1:8089`
- Active slot marker: `/opt/sub2api-deploy/.active_slot`
- Reverse proxy: nginx `server_name ai.lixnus.cc`

The nginx upstream keeps one local app slot as primary and the other as backup:

```nginx
upstream sub2api_backend {
    server 127.0.0.1:8088 max_fails=2 fail_timeout=10s;
    server 127.0.0.1:8089 backup max_fails=2 fail_timeout=10s;
    keepalive 32;
}
```

## Recommended Update Method

Use the local artifact deployment script:

```bash
SSHPASS='server-password' ./deploy/local-artifact-deploy.sh
```

If SSH key login is configured, omit `SSHPASS`:

```bash
./deploy/local-artifact-deploy.sh
```

This is the preferred flow for normal updates:

1. Build the frontend locally.
2. Build the Linux `amd64` backend binary locally with embedded frontend assets.
3. Package only the runtime artifact:
   - `sub2api`
   - `resources/`
   - `docker-entrypoint.sh`
   - lightweight artifact `Dockerfile`
4. Upload the artifact to `/opt/sub2api-artifact`.
5. On the server, build a small image from the existing `sub2api-local:latest` base and copy `pg_dump`/`psql` from local `postgres:18-alpine`.
6. Update the idle app slot first.
7. Check the idle slot health.
8. Switch nginx primary to the healthy slot.
9. Refresh the former active slot with the same image.
10. Verify public `/health`.

The server does not compile frontend or Go source during this flow.

## One-Command Script

Script:

```bash
deploy/local-artifact-deploy.sh
```

Common usage:

```bash
# Full local build, upload, dual-slot update, and public health verification.
SSHPASS='server-password' ./deploy/local-artifact-deploy.sh

# Use SSH key auth.
./deploy/local-artifact-deploy.sh

# Build and upload only; do not change running containers.
SSHPASS='server-password' ./deploy/local-artifact-deploy.sh --upload-only

# Reuse the artifact already present on the server and only run remote switching.
SSHPASS='server-password' ./deploy/local-artifact-deploy.sh --remote-only
```

Useful environment overrides:

```bash
REMOTE_HOST=103.189.141.223
REMOTE_USER=root
REMOTE_PORT=22
REMOTE_ARTIFACT_DIR=/opt/sub2api-artifact
REMOTE_DEPLOY_DIR=/opt/sub2api-deploy
REMOTE_NGINX_CONF=/etc/nginx/sites-enabled/ai.lixnus.cc
PUBLIC_HEALTH_URL=https://ai.lixnus.cc/health
BUILD_FRONTEND=1
VERIFY_PUBLIC=1
```

## Secret Handling

Secrets are stored on the server, primarily in:

- `/opt/sub2api-deploy/config.yaml`
- container environment variables
- database volumes

Do not copy passwords, tokens, JWT secrets, database passwords, OAuth client secrets, or full config contents into this document or into Git.

The local deploy script does not store the server password. Use one of these:

```bash
SSHPASS='server-password' ./deploy/local-artifact-deploy.sh
```

or configure SSH key login and run:

```bash
./deploy/local-artifact-deploy.sh
```

## Health Checks

Public check:

```bash
curl -fsS https://ai.lixnus.cc/health
```

Server checks:

```bash
cat /opt/sub2api-deploy/.active_slot
docker ps --format 'table {{.Names}}\t{{.Image}}\t{{.Status}}' | grep -E 'sub2api(_a|_b|-postgres|-redis)|NAMES'
curl -fsS http://127.0.0.1:8088/health
curl -fsS http://127.0.0.1:8089/health
```

Expected public health response:

```json
{"status":"ok"}
```

## Manual Fallback

If the one-command script cannot be used, keep the same deployment shape:

```bash
# Local
pnpm --dir frontend run build

VERSION_VALUE="$(tr -d '\r\n' < backend/cmd/server/VERSION)"
COMMIT_VALUE="$(git rev-parse --short=8 HEAD)"
DATE_VALUE="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
ARTIFACT_DIR="/tmp/sub2api-deploy-artifact"

rm -rf "$ARTIFACT_DIR"
mkdir -p "$ARTIFACT_DIR"

cd backend
GOTOOLCHAIN=auto CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
  -tags embed \
  -ldflags="-s -w -X main.Version=${VERSION_VALUE} -X main.Commit=${COMMIT_VALUE} -X main.Date=${DATE_VALUE} -X main.BuildType=release" \
  -trimpath \
  -o "${ARTIFACT_DIR}/sub2api" \
  ./cmd/server
cd ..

cp -R backend/resources "${ARTIFACT_DIR}/resources"
cp deploy/docker-entrypoint.sh "${ARTIFACT_DIR}/docker-entrypoint.sh"
chmod +x "${ARTIFACT_DIR}/sub2api" "${ARTIFACT_DIR}/docker-entrypoint.sh"
```

Create `${ARTIFACT_DIR}/Dockerfile`:

```Dockerfile
FROM postgres:18-alpine AS pg-client
FROM sub2api-local:latest
USER root
COPY --from=pg-client /usr/local/bin/pg_dump /usr/local/bin/pg_dump
COPY --from=pg-client /usr/local/bin/psql /usr/local/bin/psql
COPY --from=pg-client /usr/local/lib/libpq.so.5* /usr/local/lib/
COPY --from=pg-client /usr/lib/libzstd.so.1 /usr/lib/
COPY --from=pg-client /usr/lib/liblz4.so.1 /usr/lib/
COPY --from=pg-client /usr/lib/libssl.so.3 /usr/lib/
COPY --from=pg-client /usr/lib/libcrypto.so.3 /usr/lib/
COPY --from=pg-client /usr/lib/libgssapi_krb5.so.2 /usr/lib/
COPY --from=pg-client /usr/lib/libldap.so.2 /usr/lib/
COPY --from=pg-client /usr/lib/libkrb5.so.3 /usr/lib/
COPY --from=pg-client /usr/lib/libk5crypto.so.3 /usr/lib/
COPY --from=pg-client /usr/lib/libcom_err.so.2 /usr/lib/
COPY --from=pg-client /usr/lib/libkrb5support.so.0 /usr/lib/
COPY --from=pg-client /usr/lib/liblber.so.2 /usr/lib/
COPY --from=pg-client /usr/lib/libsasl2.so.3 /usr/lib/
COPY --from=pg-client /usr/lib/libkeyutils.so.1 /usr/lib/
COPY --from=pg-client /usr/lib/libedit.so.0 /usr/lib/
COPY --from=pg-client /usr/lib/libncursesw.so.6 /usr/lib/
COPY --chown=sub2api:sub2api sub2api /app/sub2api
COPY --chown=sub2api:sub2api resources /app/resources
COPY docker-entrypoint.sh /app/docker-entrypoint.sh
RUN chmod +x /app/sub2api /app/docker-entrypoint.sh /usr/local/bin/pg_dump /usr/local/bin/psql \
    && pg_dump --version \
    && psql --version
ENTRYPOINT ["/app/docker-entrypoint.sh"]
CMD ["/app/sub2api"]
```

Upload:

```bash
ssh root@103.189.141.223 'rm -rf /opt/sub2api-artifact && mkdir -p /opt/sub2api-artifact'
rsync -az --delete "${ARTIFACT_DIR}/" root@103.189.141.223:/opt/sub2api-artifact/
```

Then prefer running the remote-only half of the script:

```bash
SSHPASS='server-password' ./deploy/local-artifact-deploy.sh --remote-only
```

## Notes

- Do not run `docker compose down` during normal updates; that can stop PostgreSQL and Redis.
- Do not build the full project image on the production server for routine updates.
- Keep `sub2api_a` and `sub2api_b` on the same final image after each successful deployment.
- If a slot fails health checks, nginx should remain pointed at the previous healthy slot.
