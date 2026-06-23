# 服务器更新部署说明

本文档说明当前香港服务器上的 Sub2API 如何更新前后端。

当前推荐方式：**优先在本地构建 Linux 二进制并上传服务器，再用服务器现有 Docker 镜像替换应用文件并重启容器**。

## 当前服务器约定

- 服务器：`root@103.79.184.134`
- 域名：`https://ai.lixnus.cc`
- 本地源码目录：`/Users/zhaolin/Documents/Lin/app/sub2api`
- 服务器源码目录：`/opt/sub2api-src`
- 服务器部署目录：`/opt/sub2api-deploy`
- Compose 文件：`/opt/sub2api-deploy/docker-compose.local.yml`
- 应用镜像：`sub2api-local:latest`
- 应用容器：`sub2api`
- 数据库容器：`sub2api-postgres`
- Redis 容器：`sub2api-redis`
- 内部健康检查：`http://127.0.0.1:8088/health`

## 为什么优先本地构建

这个项目的前端会通过 `-tags embed` 嵌入 Go 后端二进制，所以只要前端或后端有代码变化，最终都需要更新后端二进制。

优先使用本地构建二进制上传，原因是：

- 不依赖服务器拉取 Go/npm 依赖，避免国内或服务器网络波动。
- 不依赖 Docker Hub 拉基础镜像。
- 只替换应用容器，不动 PostgreSQL 和 Redis。
- 前端改动也能一次性嵌入后端二进制，部署结果更稳定。

服务器 Git 拉取 + Docker 构建仍保留为备用方案，适合服务器网络正常、希望完全在服务器构建时使用。

## 推荐更新流程

### 1. 本地确认代码状态

```bash
cd /Users/zhaolin/Documents/Lin/app/sub2api
git status --short
git pull --ff-only origin main
```

如果本地有未提交改动，先确认是否需要提交：

```bash
git add <changed-files>
git commit -m "your change message"
git push origin main
```

不需要提交的本地临时文件不要带入部署。

### 2. 本地验证

按变更范围选择验证命令。

前端相关改动：

```bash
pnpm --dir frontend typecheck
pnpm --dir frontend test:run
```

后端相关改动：

```bash
go test ./...
```

如果只是小范围改动，可以先跑对应的定向测试。

### 3. 构建前端

```bash
cd /Users/zhaolin/Documents/Lin/app/sub2api
pnpm --dir frontend run build
```

构建结果会进入后端嵌入资源目录，后续 Go 构建需要带 `-tags embed`。

### 4. 构建 Linux amd64 后端二进制

```bash
cd /Users/zhaolin/Documents/Lin/app/sub2api

VERSION_VALUE="$(tr -d '\r\n' < backend/cmd/server/VERSION)"
DATE_VALUE="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
ARTIFACT_DIR="/tmp/sub2api-deploy-artifact"

rm -rf "$ARTIFACT_DIR"
mkdir -p "$ARTIFACT_DIR"

cd backend
GOTOOLCHAIN=auto CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
  -tags embed \
  -ldflags="-s -w -X main.Version=${VERSION_VALUE} -X main.Commit=$(git rev-parse --short HEAD) -X main.Date=${DATE_VALUE} -X main.BuildType=release" \
  -trimpath \
  -o "$ARTIFACT_DIR/sub2api" \
  ./cmd/server
```

### 5. 打包运行资源

```bash
cd /Users/zhaolin/Documents/Lin/app/sub2api

ARTIFACT_DIR="/tmp/sub2api-deploy-artifact"
cp -R backend/resources "$ARTIFACT_DIR/resources"
cp deploy/docker-entrypoint.sh "$ARTIFACT_DIR/docker-entrypoint.sh"

cat >"$ARTIFACT_DIR/Dockerfile" <<'EOF'
FROM sub2api-local:latest
USER root
COPY --chown=sub2api:sub2api sub2api /app/sub2api
COPY --chown=sub2api:sub2api resources /app/resources
COPY docker-entrypoint.sh /app/docker-entrypoint.sh
RUN chmod +x /app/sub2api /app/docker-entrypoint.sh
ENTRYPOINT ["/app/docker-entrypoint.sh"]
CMD ["/app/sub2api"]
EOF
```

这个 Dockerfile 使用服务器已有的 `sub2api-local:latest` 作为基础，只替换新的应用二进制和资源。

### 6. 同步源码到服务器

这一步用于让服务器保留一份最新源码，方便排查、对照版本和后续备用构建。

```bash
cd /Users/zhaolin/Documents/Lin/app/sub2api

rsync -az --delete --stats \
  --exclude='.git/' \
  --exclude='node_modules/' \
  --exclude='frontend/node_modules/' \
  --exclude='frontend/dist/' \
  --exclude='backend/internal/web/dist/' \
  --exclude='.DS_Store' \
  --exclude='generated-images/' \
  --exclude='generated-videos/' \
  ./ root@103.79.184.134:/opt/sub2api-src/
```

### 7. 上传部署产物

```bash
ssh root@103.79.184.134 'rm -rf /opt/sub2api-artifact && mkdir -p /opt/sub2api-artifact'
rsync -az --delete /tmp/sub2api-deploy-artifact/ root@103.79.184.134:/opt/sub2api-artifact/
```

### 8. 服务器构建轻量镜像并重启应用容器

```bash
ssh root@103.79.184.134 '
set -e
cd /opt/sub2api-artifact
docker build -t sub2api-local:latest .

cd /opt/sub2api-deploy
docker compose -f docker-compose.local.yml up -d --no-deps --force-recreate sub2api
'
```

注意：这里使用 `--no-deps --force-recreate sub2api`，只重建应用容器，不重启数据库和 Redis。

### 9. 回测

```bash
ssh root@103.79.184.134 '
set -e
docker inspect -f "{{.State.Health.Status}} {{.Image}}" sub2api
curl -fsS http://127.0.0.1:8088/health
cd /opt/sub2api-deploy
docker compose -f docker-compose.local.yml ps
docker compose -f docker-compose.local.yml logs --tail=80 sub2api
'

curl -I https://ai.lixnus.cc/
curl -fsS https://ai.lixnus.cc/api/v1/settings/public
```

成功标准：

- `sub2api` 容器状态为 `healthy`
- 内部健康检查返回成功
- `https://ai.lixnus.cc/` 返回 `200`
- 关键功能按本次改动范围完成页面或接口回测

## 一键命令模板

确认本地代码已经是要部署的版本后，可以按下面模板执行。

```bash
cd /Users/zhaolin/Documents/Lin/app/sub2api

pnpm --dir frontend run build

VERSION_VALUE="$(tr -d '\r\n' < backend/cmd/server/VERSION)"
DATE_VALUE="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
COMMIT_VALUE="$(git rev-parse --short HEAD)"
ARTIFACT_DIR="/tmp/sub2api-deploy-artifact"

rm -rf "$ARTIFACT_DIR"
mkdir -p "$ARTIFACT_DIR"

cd backend
GOTOOLCHAIN=auto CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
  -tags embed \
  -ldflags="-s -w -X main.Version=${VERSION_VALUE} -X main.Commit=${COMMIT_VALUE} -X main.Date=${DATE_VALUE} -X main.BuildType=release" \
  -trimpath \
  -o "$ARTIFACT_DIR/sub2api" \
  ./cmd/server

cd /Users/zhaolin/Documents/Lin/app/sub2api
cp -R backend/resources "$ARTIFACT_DIR/resources"
cp deploy/docker-entrypoint.sh "$ARTIFACT_DIR/docker-entrypoint.sh"
cat >"$ARTIFACT_DIR/Dockerfile" <<'EOF'
FROM sub2api-local:latest
USER root
COPY --chown=sub2api:sub2api sub2api /app/sub2api
COPY --chown=sub2api:sub2api resources /app/resources
COPY docker-entrypoint.sh /app/docker-entrypoint.sh
RUN chmod +x /app/sub2api /app/docker-entrypoint.sh
ENTRYPOINT ["/app/docker-entrypoint.sh"]
CMD ["/app/sub2api"]
EOF

rsync -az --delete --stats \
  --exclude='.git/' \
  --exclude='node_modules/' \
  --exclude='frontend/node_modules/' \
  --exclude='frontend/dist/' \
  --exclude='backend/internal/web/dist/' \
  --exclude='.DS_Store' \
  --exclude='generated-images/' \
  --exclude='generated-videos/' \
  ./ root@103.79.184.134:/opt/sub2api-src/

ssh root@103.79.184.134 'rm -rf /opt/sub2api-artifact && mkdir -p /opt/sub2api-artifact'
rsync -az --delete "$ARTIFACT_DIR/" root@103.79.184.134:/opt/sub2api-artifact/

ssh root@103.79.184.134 '
set -e
cd /opt/sub2api-artifact
docker build -t sub2api-local:latest .
cd /opt/sub2api-deploy
docker compose -f docker-compose.local.yml up -d --no-deps --force-recreate sub2api
docker inspect -f "{{.State.Health.Status}}" sub2api
curl -fsS http://127.0.0.1:8088/health
'
```

## 备用方案：服务器 Git + Docker 构建

如果服务器网络正常，也可以使用一键更新脚本。

首次安装：

```bash
ssh root@103.79.184.134 'install -m 755 /opt/sub2api-src/deploy/update-server.sh /usr/local/bin/sub2api-update'
```

日常执行：

```bash
ssh root@103.79.184.134 'sub2api-update --detach'
```

查看日志：

```bash
ssh root@103.79.184.134 '
ls -lt /opt/sub2api-deploy/update-logs/
tail -f /opt/sub2api-deploy/update-logs/update-YYYYMMDD-HHMMSS.log
'
```

备用方案会在服务器上拉取 GitHub 最新代码并执行 Docker build。它更自动，但受服务器访问 GitHub、npm、Go 依赖源和 Docker Hub 的影响更大。

## 回滚

推荐更新方式会直接把 `sub2api-local:latest` 打成新镜像。更新前如需保留回滚点，可以在服务器先打 tag：

```bash
ssh root@103.79.184.134 '
ROLLBACK_TAG="sub2api-local:rollback-$(date +%Y%m%d-%H%M%S)"
docker tag sub2api-local:latest "$ROLLBACK_TAG"
echo "$ROLLBACK_TAG"
'
```

需要回滚时：

```bash
ssh root@103.79.184.134 '
set -e
docker tag sub2api-local:rollback-YYYYMMDD-HHMMSS sub2api-local:latest
cd /opt/sub2api-deploy
docker compose -f docker-compose.local.yml up -d --no-deps --force-recreate sub2api
curl -fsS http://127.0.0.1:8088/health
'
```

## 常用排查命令

查看容器状态：

```bash
ssh root@103.79.184.134 'cd /opt/sub2api-deploy && docker compose -f docker-compose.local.yml ps'
```

查看应用日志：

```bash
ssh root@103.79.184.134 'cd /opt/sub2api-deploy && docker compose -f docker-compose.local.yml logs --tail=120 sub2api'
```

查看最近镜像：

```bash
ssh root@103.79.184.134 'docker images sub2api-local --format "table {{.Repository}}\t{{.Tag}}\t{{.ID}}\t{{.CreatedSince}}"'
```

查看服务器源码版本：

```bash
ssh root@103.79.184.134 'cd /opt/sub2api-src && git log -1 --oneline 2>/dev/null || true'
```

## 注意事项

- 不要运行 `docker compose down`，除非明确要停止 PostgreSQL 和 Redis。
- `.env`、数据库数据、备份文件、密钥和 token 不要提交到 Git。
- 前端改动也需要重新构建 Go 二进制，因为前端资源会嵌入后端。
- 上传产物时不要把 `node_modules`、生成图片、生成视频等本地临时目录同步到服务器。
- 每次部署后至少检查容器健康状态和 `https://ai.lixnus.cc/` 首页。
