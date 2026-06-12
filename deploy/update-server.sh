#!/usr/bin/env bash
# One-command updater for the self-hosted Sub2API Docker deployment.

set -euo pipefail

SRC_DIR="${SRC_DIR:-/opt/sub2api-src}"
DEPLOY_DIR="${DEPLOY_DIR:-/opt/sub2api-deploy}"
COMPOSE_FILE="${COMPOSE_FILE:-docker-compose.local.yml}"
SERVICE_NAME="${SERVICE_NAME:-sub2api}"
IMAGE_NAME="${IMAGE_NAME:-sub2api-local:latest}"
REMOTE="${REMOTE:-origin}"
BRANCH="${BRANCH:-main}"
HEALTH_URL="${HEALTH_URL:-http://127.0.0.1:8088/health}"
HEALTH_RETRIES="${HEALTH_RETRIES:-30}"
HEALTH_INTERVAL="${HEALTH_INTERVAL:-2}"
GOPROXY="${GOPROXY:-https://goproxy.cn,direct}"
GOSUMDB="${GOSUMDB:-sum.golang.google.cn}"
LOG_DIR="${LOG_DIR:-${DEPLOY_DIR}/update-logs}"
LOCK_DIR="${LOCK_DIR:-/tmp/sub2api-update.lock}"

DETACH=false
FORCE=false
NO_PULL=false
SKIP_BUILD=false
NO_ROLLBACK=false
ARGS_FOR_CHILD=()

usage() {
	cat <<'USAGE'
Usage: deploy/update-server.sh [options]

Options:
  --detach       Run update in background and write logs under update-logs/.
  --force        Build and restart even when no relevant source changes are found.
  --no-pull      Do not fetch/reset from Git before building.
  --skip-build   Restart the compose service with the current local image.
  --no-rollback  Do not retag the previous image if health check fails.
  -h, --help     Show this help message.

Environment overrides:
  SRC_DIR=/opt/sub2api-src
  DEPLOY_DIR=/opt/sub2api-deploy
  COMPOSE_FILE=docker-compose.local.yml
  IMAGE_NAME=sub2api-local:latest
  SERVICE_NAME=sub2api
  REMOTE=origin
  BRANCH=main
  HEALTH_URL=http://127.0.0.1:8088/health
USAGE
}

while [ "$#" -gt 0 ]; do
	case "$1" in
		--detach)
			DETACH=true
			;;
		--force)
			FORCE=true
			ARGS_FOR_CHILD+=("$1")
			;;
		--no-pull)
			NO_PULL=true
			ARGS_FOR_CHILD+=("$1")
			;;
		--skip-build)
			SKIP_BUILD=true
			ARGS_FOR_CHILD+=("$1")
			;;
		--no-rollback)
			NO_ROLLBACK=true
			ARGS_FOR_CHILD+=("$1")
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

mkdir -p "$LOG_DIR"
RUN_ID="$(date +%Y%m%d-%H%M%S)"
LOG_FILE="${LOG_DIR}/update-${RUN_ID}.log"

if [ "$DETACH" = true ] && [ "${SUB2API_UPDATE_DETACHED:-}" != "1" ]; then
	echo "Starting Sub2API update in background."
	echo "Log: ${LOG_FILE}"
	SUB2API_UPDATE_DETACHED=1 nohup "$0" "${ARGS_FOR_CHILD[@]}" >"$LOG_FILE" 2>&1 &
	echo "PID: $!"
	exit 0
fi

if [ "${SUB2API_UPDATE_DETACHED:-}" != "1" ]; then
	exec > >(tee -a "$LOG_FILE") 2>&1
fi

log() {
	printf '[%s] %s\n' "$(date '+%Y-%m-%d %H:%M:%S')" "$*"
}

fail() {
	log "ERROR: $*"
	exit 1
}

cleanup_lock() {
	rmdir "$LOCK_DIR" 2>/dev/null || true
}

compose() {
	cd "$DEPLOY_DIR"
	if docker compose version >/dev/null 2>&1; then
		docker compose -f "$COMPOSE_FILE" "$@"
	else
		docker-compose -f "$COMPOSE_FILE" "$@"
	fi
}

is_relevant_change() {
	awk '
		/^backend\// { found=1 }
		/^frontend\// { found=1 }
		/^deploy\/docker-entrypoint\.sh$/ { found=1 }
		/^deploy\/docker-compose/ { found=1 }
		/^docs\/legal\// { found=1 }
		/^Dockerfile$/ { found=1 }
		/^\.dockerignore$/ { found=1 }
		/^go\.mod$/ { found=1 }
		/^go\.sum$/ { found=1 }
		END { exit found ? 0 : 1 }
	'
}

wait_for_health() {
	local i
	for i in $(seq 1 "$HEALTH_RETRIES"); do
		if curl -fsS --max-time 5 "$HEALTH_URL" >/dev/null; then
			log "Health check passed: ${HEALTH_URL}"
			return 0
		fi
		log "Health check not ready (${i}/${HEALTH_RETRIES}); waiting ${HEALTH_INTERVAL}s..."
		sleep "$HEALTH_INTERVAL"
	done
	return 1
}

rollback_image() {
	local previous_image_id="$1"
	if [ "$NO_ROLLBACK" = true ]; then
		log "Rollback disabled by --no-rollback."
		return 1
	fi
	if [ -z "$previous_image_id" ]; then
		log "No previous image id recorded; rollback unavailable."
		return 1
	fi
	log "Rolling back ${IMAGE_NAME} to previous image ${previous_image_id}."
	docker tag "$previous_image_id" "$IMAGE_NAME"
	compose up -d "$SERVICE_NAME"
	wait_for_health
}

main() {
	log "Sub2API update started."
	log "Log file: ${LOG_FILE}"

	if ! mkdir "$LOCK_DIR" 2>/dev/null; then
		fail "another update appears to be running: ${LOCK_DIR}"
	fi
	trap cleanup_lock EXIT

	[ -d "$SRC_DIR/.git" ] || fail "source directory is not a Git repository: ${SRC_DIR}"
	[ -d "$DEPLOY_DIR" ] || fail "deploy directory does not exist: ${DEPLOY_DIR}"
	[ -f "${DEPLOY_DIR}/${COMPOSE_FILE}" ] || fail "compose file not found: ${DEPLOY_DIR}/${COMPOSE_FILE}"

	cd "$SRC_DIR"
	if [ -n "$(git status --porcelain)" ]; then
		fail "source tree has local changes; commit/stash them before updating"
	fi

	local old_rev new_rev changed_files build_needed previous_image_id
	old_rev="$(git rev-parse HEAD)"
	log "Current revision: ${old_rev}"

	if [ "$NO_PULL" = false ]; then
		log "Fetching ${REMOTE}/${BRANCH}..."
		git fetch "$REMOTE" "$BRANCH"
		new_rev="$(git rev-parse "${REMOTE}/${BRANCH}")"
		if [ "$old_rev" != "$new_rev" ]; then
			log "Updating source to ${new_rev}."
			git reset --hard "${REMOTE}/${BRANCH}"
		else
			log "Source is already up to date."
		fi
	else
		new_rev="$old_rev"
		log "Skipping Git fetch/reset because --no-pull was set."
	fi

	changed_files=""
	if [ "$old_rev" != "$new_rev" ]; then
		changed_files="$(git diff --name-only "$old_rev" "$new_rev")"
	fi

	build_needed=false
	if [ "$SKIP_BUILD" = true ]; then
		log "Skipping Docker build because --skip-build was set."
	elif [ "$FORCE" = true ]; then
		build_needed=true
	elif [ "$old_rev" = "$new_rev" ]; then
		log "No new revision. Use --force to rebuild anyway."
	else
		if printf '%s\n' "$changed_files" | is_relevant_change; then
			build_needed=true
		else
			log "No backend/frontend/deployment changes detected; skipping build."
			log "Changed files:"
			printf '%s\n' "$changed_files"
		fi
	fi

	previous_image_id="$(docker images -q "$IMAGE_NAME" 2>/dev/null | head -n 1 || true)"
	log "Previous image id: ${previous_image_id:-none}"

	if [ "$build_needed" = true ]; then
		log "Building ${IMAGE_NAME}."
		docker build \
			--build-arg "GOPROXY=${GOPROXY}" \
			--build-arg "GOSUMDB=${GOSUMDB}" \
			-t "$IMAGE_NAME" \
			"$SRC_DIR"
	else
		log "Docker build skipped."
	fi

	log "Starting compose service ${SERVICE_NAME}."
	compose up -d "$SERVICE_NAME"

	if ! wait_for_health; then
		log "Health check failed after update."
		if rollback_image "$previous_image_id"; then
			fail "update failed; rolled back to previous image"
		fi
		fail "update failed; rollback was not completed"
	fi

	log "Container status:"
	compose ps "$SERVICE_NAME"
	log "Latest revision: $(git -C "$SRC_DIR" log -1 --oneline)"
	log "Sub2API update completed successfully."
}

main "$@"
