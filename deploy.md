# Deployment Runbook

## Last Deployment

- Date: 2026-06-14
- Project: sub2api
- Target server: `root@103.79.184.134:22`
- Public URL: `https://ai.lixnus.cc/`
- Source path on server: `/opt/sub2api-src`
- Runtime path on server: `/opt/sub2api-deploy`
- Runtime: Docker Compose, app image `sub2api-local:latest`
- App container: `sub2api`
- Database/cache containers: `sub2api-postgres`, `sub2api-redis`
- Internal app port: `127.0.0.1:8088 -> 8080`
- Reverse proxy: nginx `server_name ai.lixnus.cc`

## Secret Handling

Secrets are stored on the server in `/opt/sub2api-deploy/.env`.
Do not copy passwords, tokens, JWT secrets, database passwords, OAuth client secrets, or full `.env` contents into this file.

## Deployment Performed

The server could not reach Docker Hub during a full Docker build:

```bash
failed to resolve source metadata for docker.io/library/alpine:3.21
```

To avoid changing Docker registry settings or touching persistent data, this deployment reused the existing local runtime image as the base and replaced only the compiled application artifact.

Steps used:

```bash
# Local verification before deployment
pnpm --dir frontend test:run src/views/admin/__tests__/UsersView.spec.ts
pnpm --dir frontend typecheck

# Sync current workspace to the server source directory
SSHPASS=... rsync -az --delete --stats \
  --exclude='.git/' \
  --exclude='node_modules/' \
  --exclude='frontend/node_modules/' \
  --exclude='frontend/dist/' \
  --exclude='backend/internal/web/dist/' \
  --exclude='.DS_Store' \
  -e 'sshpass -e ssh -p 22 -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null' \
  ./ root@103.79.184.134:/opt/sub2api-src/

# Build frontend locally into backend/internal/web/dist
pnpm --dir frontend run build

# Build Linux amd64 backend locally with embedded frontend
VERSION_VALUE="$(tr -d '\r\n' < backend/cmd/server/VERSION)"
DATE_VALUE="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
mkdir -p /tmp/sub2api-deploy-artifact
cd backend
GOTOOLCHAIN=auto CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
  -tags embed \
  -ldflags="-s -w -X main.Version=${VERSION_VALUE} -X main.Commit=local-deploy -X main.Date=${DATE_VALUE} -X main.BuildType=release" \
  -trimpath \
  -o /tmp/sub2api-deploy-artifact/sub2api \
  ./cmd/server

# Package runtime resources
cp -R backend/resources /tmp/sub2api-deploy-artifact/resources
cp deploy/docker-entrypoint.sh /tmp/sub2api-deploy-artifact/docker-entrypoint.sh

# Upload artifact
ssh root@103.79.184.134 'rm -rf /opt/sub2api-artifact && mkdir -p /opt/sub2api-artifact'
rsync -az --delete /tmp/sub2api-deploy-artifact/ root@103.79.184.134:/opt/sub2api-artifact/

# Build a small image from the existing runtime image, then recreate app only
cat >/opt/sub2api-artifact/Dockerfile <<'EOF'
FROM sub2api-local:latest
USER root
COPY --chown=sub2api:sub2api sub2api /app/sub2api
COPY --chown=sub2api:sub2api resources /app/resources
COPY docker-entrypoint.sh /app/docker-entrypoint.sh
RUN chmod +x /app/sub2api /app/docker-entrypoint.sh
ENTRYPOINT ["/app/docker-entrypoint.sh"]
CMD ["/app/sub2api"]
EOF

cd /opt/sub2api-artifact
docker build -t sub2api-local:latest .

cd /opt/sub2api-deploy
docker compose -f docker-compose.local.yml up -d --no-deps --force-recreate sub2api
```

## Verification

Commands:

```bash
docker inspect -f '{{.State.Health.Status}} {{.Image}}' sub2api
curl -s -o /dev/null -w '%{http_code}' http://127.0.0.1:8088/health
curl -I https://ai.lixnus.cc/
```

Results:

- Container health: `healthy`
- Internal health endpoint: `200`
- Public homepage: `https://ai.lixnus.cc/` returned `200 OK`
- Direct IP access returns nginx `404` because nginx is configured for `server_name ai.lixnus.cc`.
- Public `/health` returns application `404`; use the internal Docker health endpoint for health checks.

## Notes For Next Deployment

- Prefer the normal full Docker build if Docker Hub connectivity is restored:

```bash
cd /opt/sub2api-src
DOCKER_BUILDKIT=1 docker build -t sub2api-local:latest .
cd /opt/sub2api-deploy
docker compose -f docker-compose.local.yml up -d --no-deps --force-recreate sub2api
```

- If Docker Hub is still unreachable, reuse the artifact replacement flow above.
- Do not run `docker compose down` unless intentionally stopping PostgreSQL and Redis; the normal app-only refresh is `up -d --no-deps --force-recreate sub2api`.
