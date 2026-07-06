#!/usr/bin/env bash
# Build Sub2API locally, upload the runtime artifact, and switch the production
# server with the lightweight dual-slot deployment flow.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

REMOTE_HOST="${REMOTE_HOST:-103.189.141.223}"
REMOTE_USER="${REMOTE_USER:-root}"
REMOTE_PORT="${REMOTE_PORT:-22}"
REMOTE_ARTIFACT_DIR="${REMOTE_ARTIFACT_DIR:-/opt/sub2api-artifact}"
REMOTE_DEPLOY_DIR="${REMOTE_DEPLOY_DIR:-/opt/sub2api-deploy}"
REMOTE_NGINX_CONF="${REMOTE_NGINX_CONF:-/etc/nginx/sites-enabled/ai.lixnus.cc}"
PUBLIC_HEALTH_URL="${PUBLIC_HEALTH_URL:-https://ai.lixnus.cc/health}"
IMAGE_REPO="${IMAGE_REPO:-sub2api-local}"
BASE_IMAGE="${BASE_IMAGE:-sub2api-local:latest}"
HEALTH_RETRIES="${HEALTH_RETRIES:-45}"
HEALTH_INTERVAL="${HEALTH_INTERVAL:-2}"
BUILD_FRONTEND="${BUILD_FRONTEND:-1}"
VERIFY_PUBLIC="${VERIFY_PUBLIC:-1}"
UPLOAD_ONLY=0
REMOTE_ONLY=0
KEEP_ARTIFACT=0

RUN_ID="$(date +%Y%m%d-%H%M%S)"
ARTIFACT_DIR="${ARTIFACT_DIR:-/tmp/sub2api-deploy-artifact-${RUN_ID}}"

usage() {
	cat <<'USAGE'
Usage: deploy/local-artifact-deploy.sh [options]

Builds the frontend and Linux amd64 backend locally, uploads only the runtime
artifact to the server, then performs a lightweight Docker image replacement
with the sub2api_a/sub2api_b dual-slot flow.

Options:
  --upload-only       Build and upload artifact, but do not update containers.
  --remote-only       Skip local build/upload and deploy the artifact already
                      present in REMOTE_ARTIFACT_DIR.
  --keep-artifact     Keep the local artifact directory after success.
  -h, --help          Show this help message.

Environment overrides:
  REMOTE_HOST=103.189.141.223
  REMOTE_USER=root
  REMOTE_PORT=22
  SSHPASS='password'              # optional; SSH key auth also works
  REMOTE_ARTIFACT_DIR=/opt/sub2api-artifact
  REMOTE_DEPLOY_DIR=/opt/sub2api-deploy
  REMOTE_NGINX_CONF=/etc/nginx/sites-enabled/ai.lixnus.cc
  PUBLIC_HEALTH_URL=https://ai.lixnus.cc/health
  IMAGE_REPO=sub2api-local
  BASE_IMAGE=sub2api-local:latest
  BUILD_FRONTEND=0                # skip pnpm frontend build when dist is ready
  VERIFY_PUBLIC=0                 # skip final public health curl
USAGE
}

while [ "$#" -gt 0 ]; do
	case "$1" in
		--upload-only)
			UPLOAD_ONLY=1
			;;
		--remote-only)
			REMOTE_ONLY=1
			;;
		--keep-artifact)
			KEEP_ARTIFACT=1
			;;
		-h|--help)
			usage
			exit 0
			;;
		*)
			echo "Unknown option: $1" >&2
			usage >&2
			exit 2
			;;
	esac
	shift
done

log() {
	printf '[%s] %s\n' "$(date '+%Y-%m-%d %H:%M:%S')" "$*"
}

fail() {
	log "ERROR: $*"
	exit 1
}

require_cmd() {
	command -v "$1" >/dev/null 2>&1 || fail "missing command: $1"
}

ssh_base() {
	ssh -p "$REMOTE_PORT" \
		-o StrictHostKeyChecking=no \
		-o UserKnownHostsFile=/dev/null \
		"$REMOTE_USER@$REMOTE_HOST" "$@"
}

ssh_cmd() {
	if [ -n "${SSHPASS:-}" ]; then
		require_cmd sshpass
		SSHPASS="$SSHPASS" sshpass -e ssh -p "$REMOTE_PORT" \
			-o StrictHostKeyChecking=no \
			-o UserKnownHostsFile=/dev/null \
			"$REMOTE_USER@$REMOTE_HOST" "$@"
	else
		ssh_base "$@"
	fi
}

rsync_cmd() {
	if [ -n "${SSHPASS:-}" ]; then
		require_cmd sshpass
		SSHPASS="$SSHPASS" sshpass -e rsync "$@"
	else
		rsync "$@"
	fi
}

sha256_file() {
	if command -v shasum >/dev/null 2>&1; then
		shasum -a 256 "$1" | awk '{print $1}'
	elif command -v sha256sum >/dev/null 2>&1; then
		sha256sum "$1" | awk '{print $1}'
	else
		fail "missing shasum or sha256sum"
	fi
}

cleanup() {
	if [ "$KEEP_ARTIFACT" = "0" ] && [ "$REMOTE_ONLY" = "0" ]; then
		rm -rf "$ARTIFACT_DIR"
	fi
}
trap cleanup EXIT

build_artifact() {
	require_cmd git
	require_cmd go
	require_cmd pnpm

	cd "$REPO_ROOT"
	[ -f backend/cmd/server/VERSION ] || fail "backend/cmd/server/VERSION not found"

	if [ "$BUILD_FRONTEND" = "1" ]; then
		log "Building frontend locally."
		pnpm --dir frontend run build
	else
		log "Skipping frontend build because BUILD_FRONTEND=0."
	fi

	local version commit build_date binary_hash
	version="$(tr -d '\r\n' < backend/cmd/server/VERSION)"
	commit="$(git rev-parse --short=8 HEAD 2>/dev/null || echo local)"
	build_date="$(date -u +%Y-%m-%dT%H:%M:%SZ)"

	log "Building Linux amd64 backend locally: version=${version}, commit=${commit}."
	rm -rf "$ARTIFACT_DIR"
	mkdir -p "$ARTIFACT_DIR"
	(
		cd backend
		GOTOOLCHAIN=auto CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
			-tags embed \
			-ldflags="-s -w -X main.Version=${version} -X main.Commit=${commit} -X main.Date=${build_date} -X main.BuildType=release" \
			-trimpath \
			-o "${ARTIFACT_DIR}/sub2api" \
			./cmd/server
	)

	cp -R "${REPO_ROOT}/backend/resources" "${ARTIFACT_DIR}/resources"
	cp "${REPO_ROOT}/deploy/docker-entrypoint.sh" "${ARTIFACT_DIR}/docker-entrypoint.sh"
	chmod +x "${ARTIFACT_DIR}/sub2api" "${ARTIFACT_DIR}/docker-entrypoint.sh"

	cat >"${ARTIFACT_DIR}/Dockerfile" <<EOF
FROM postgres:18-alpine AS pg-client
FROM ${BASE_IMAGE}
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
RUN chmod +x /app/sub2api /app/docker-entrypoint.sh /usr/local/bin/pg_dump /usr/local/bin/psql \\
    && pg_dump --version \\
    && psql --version
ENTRYPOINT ["/app/docker-entrypoint.sh"]
CMD ["/app/sub2api"]
EOF

	binary_hash="$(sha256_file "${ARTIFACT_DIR}/sub2api")"
	log "Artifact ready: ${ARTIFACT_DIR}"
	log "Binary sha256: ${binary_hash}"
}

upload_artifact() {
	require_cmd rsync
	require_cmd ssh

	log "Preparing remote artifact directory: ${REMOTE_USER}@${REMOTE_HOST}:${REMOTE_ARTIFACT_DIR}"
	ssh_cmd "rm -rf '${REMOTE_ARTIFACT_DIR}' && mkdir -p '${REMOTE_ARTIFACT_DIR}'"

	log "Uploading artifact."
	rsync_cmd -az --delete \
		-e "ssh -p ${REMOTE_PORT} -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null" \
		"${ARTIFACT_DIR}/" \
		"${REMOTE_USER}@${REMOTE_HOST}:${REMOTE_ARTIFACT_DIR}/"
}

remote_deploy() {
	require_cmd ssh

	local version commit deploy_tag
	version="$(tr -d '\r\n' < "${REPO_ROOT}/backend/cmd/server/VERSION" 2>/dev/null || echo unknown)"
	commit="$(git -C "$REPO_ROOT" rev-parse --short=8 HEAD 2>/dev/null || echo local)"
	deploy_tag="${IMAGE_REPO}:deploy-${RUN_ID}-v${version//[^0-9A-Za-z_.-]/_}"

	log "Starting remote dual-slot deployment: ${deploy_tag}"
	ssh_cmd \
		"REMOTE_ARTIFACT_DIR='${REMOTE_ARTIFACT_DIR}' REMOTE_DEPLOY_DIR='${REMOTE_DEPLOY_DIR}' REMOTE_NGINX_CONF='${REMOTE_NGINX_CONF}' IMAGE_REPO='${IMAGE_REPO}' BASE_IMAGE='${BASE_IMAGE}' DEPLOY_TAG='${deploy_tag}' HEALTH_RETRIES='${HEALTH_RETRIES}' HEALTH_INTERVAL='${HEALTH_INTERVAL}' bash -s" <<'REMOTE_SCRIPT'
set -euo pipefail

log() {
	printf '[remote %s] %s\n' "$(date '+%Y-%m-%d %H:%M:%S')" "$*"
}

fail() {
	log "ERROR: $*"
	exit 1
}

wait_for_health() {
	local port="$1"
	local i
	for i in $(seq 1 "$HEALTH_RETRIES"); do
		if curl -fsS --max-time 5 "http://127.0.0.1:${port}/health" >/dev/null; then
			log "Slot on port ${port} is healthy."
			return 0
		fi
		log "Waiting for slot health on ${port} (${i}/${HEALTH_RETRIES})."
		sleep "$HEALTH_INTERVAL"
	done
	return 1
}

slot_port() {
	case "$1" in
		a) printf '8088' ;;
		b) printf '8089' ;;
		*) fail "invalid slot: $1" ;;
	esac
}

other_slot() {
	case "$1" in
		a) printf 'b' ;;
		b) printf 'a' ;;
		*) printf 'a' ;;
	esac
}

write_nginx_upstream() {
	local primary_port="$1"
	local backup_port="$2"
	local tmp
	tmp="$(mktemp)"
	awk -v primary="$primary_port" -v backup="$backup_port" '
		/^upstream sub2api_backend \{/ {
			print
			print "    server 127.0.0.1:" primary " max_fails=2 fail_timeout=10s;"
			print "    server 127.0.0.1:" backup " backup max_fails=2 fail_timeout=10s;"
			print "    keepalive 32;"
			in_block=1
			next
		}
		in_block && /^\}/ {
			print
			in_block=0
			next
		}
		!in_block { print }
	' "$REMOTE_NGINX_CONF" > "$tmp"
	mkdir -p "${REMOTE_DEPLOY_DIR}/nginx-backups"
	cp "$REMOTE_NGINX_CONF" "${REMOTE_DEPLOY_DIR}/nginx-backups/$(basename "$REMOTE_NGINX_CONF").bak.$(date +%Y%m%d%H%M%S)"
	cat "$tmp" > "$REMOTE_NGINX_CONF"
	rm -f "$tmp"
	nginx -t
	systemctl reload nginx
}

run_slot() {
	local slot="$1"
	local port="$2"
	local source_container="$3"
	local env_file
	env_file="$(mktemp)"

	if docker ps -a --format '{{.Names}}' | grep -qx "$source_container"; then
		docker inspect "$source_container" --format '{{range .Config.Env}}{{println .}}{{end}}' > "$env_file"
	elif docker ps -a --format '{{.Names}}' | grep -qx sub2api; then
		docker inspect sub2api --format '{{range .Config.Env}}{{println .}}{{end}}' > "$env_file"
	else
		printf 'CONFIG_PATH=/app/config.yaml\n' > "$env_file"
	fi

	docker rm -f "sub2api_${slot}" >/dev/null 2>&1 || true
	docker run -d \
		--name "sub2api_${slot}" \
		--restart unless-stopped \
		--network sub2api-deploy_default \
		--env-file "$env_file" \
		-v "${REMOTE_DEPLOY_DIR}/config.yaml:/app/config.yaml:ro" \
		-v "${REMOTE_DEPLOY_DIR}/app-data:/app/data" \
		-p "127.0.0.1:${port}:8080" \
		"$DEPLOY_TAG" >/dev/null
	rm -f "$env_file"
	wait_for_health "$port"
}

[ -d "$REMOTE_ARTIFACT_DIR" ] || fail "artifact directory not found: $REMOTE_ARTIFACT_DIR"
[ -f "${REMOTE_ARTIFACT_DIR}/Dockerfile" ] || fail "artifact Dockerfile not found"
[ -f "${REMOTE_ARTIFACT_DIR}/sub2api" ] || fail "artifact binary not found"
[ -d "$REMOTE_DEPLOY_DIR" ] || fail "deploy directory not found: $REMOTE_DEPLOY_DIR"
[ -f "$REMOTE_NGINX_CONF" ] || fail "nginx config not found: $REMOTE_NGINX_CONF"

log "Building lightweight image ${DEPLOY_TAG} from ${BASE_IMAGE}."
cd "$REMOTE_ARTIFACT_DIR"
docker image inspect "$BASE_IMAGE" >/dev/null 2>&1 || fail "base image does not exist: $BASE_IMAGE"
docker build -t "$DEPLOY_TAG" -t "${IMAGE_REPO}:latest" .
docker run --rm --entrypoint /app/sub2api "$DEPLOY_TAG" --version || fail "new binary failed to start"

active="a"
if [ -f "${REMOTE_DEPLOY_DIR}/.active_slot" ]; then
	active="$(tr -d '[:space:]' < "${REMOTE_DEPLOY_DIR}/.active_slot")"
fi
case "$active" in
	a|b) ;;
	*) active="a" ;;
esac

idle="$(other_slot "$active")"
active_port="$(slot_port "$active")"
idle_port="$(slot_port "$idle")"

log "Current active slot: ${active} (${active_port}); idle slot: ${idle} (${idle_port})."
run_slot "$idle" "$idle_port" "sub2api_${active}"

log "Switching nginx primary to slot ${idle}."
write_nginx_upstream "$idle_port" "$active_port"
printf '%s\n' "$idle" > "${REMOTE_DEPLOY_DIR}/.active_slot"
printf '%s\n' "$DEPLOY_TAG" > "${REMOTE_DEPLOY_DIR}/.last_deploy_tag"

log "Refreshing former active slot ${active} with the same image."
run_slot "$active" "$active_port" "sub2api_${idle}"

log "Dual-slot deployment completed."
docker ps --format 'table {{.Names}}\t{{.Image}}\t{{.Status}}' | grep -E 'sub2api(_a|_b|-postgres|-redis)|NAMES'
REMOTE_SCRIPT

	log "Remote deployment completed: ${deploy_tag}"
}

verify_public() {
	if [ "$VERIFY_PUBLIC" = "0" ]; then
		log "Skipping public health verification because VERIFY_PUBLIC=0."
		return
	fi
	require_cmd curl
	log "Verifying public health: ${PUBLIC_HEALTH_URL}"
	curl -fsS --max-time 10 "$PUBLIC_HEALTH_URL"
	printf '\n'
}

main() {
	if [ "$REMOTE_ONLY" = "0" ]; then
		build_artifact
		upload_artifact
	else
		log "Skipping local build/upload because --remote-only was set."
	fi

	if [ "$UPLOAD_ONLY" = "1" ]; then
		log "Upload-only mode finished. Remote containers were not changed."
		return
	fi

	remote_deploy
	verify_public
	log "Deployment finished successfully."
}

main "$@"
