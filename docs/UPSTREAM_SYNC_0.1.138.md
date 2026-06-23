# Sub2API 0.1.138 同步升级记录

本文档记录本次从官方 Sub2API 同步到 `0.1.138`、推送 fork、并更新生产服务器的完整过程，后续升级官方版本时可按此流程复用。

## 本次结果

- 本地分支：`main`
- Fork 远程：`origin ssh://git@ssh.github.com:443/zLincna/sub2api.git`
- 官方远程：`upstream https://github.com/Wei-Shaw/sub2api.git`
- 同步目标：官方 `v0.1.138`，并合并官方 `main` 上版本号同步提交
- 最终本地/fork 提交：`b54b794b`
- 服务器版本：`Sub2API 0.1.138 (commit: b54b794b)`
- 生产域名：`https://ai.lixnus.cc`
- 服务器：`root@103.79.184.134`
- 部署方式：本地构建 Linux 二进制并上传服务器，轻量重建 `sub2api-local:latest`

## 本次合并提交

```bash
b54b794b test: mock settings route in admin settings specs
b594f523 merge upstream main after v0.1.138
bd117162 merge upstream v0.1.138
0670fcfa docs: prefer local binary server updates
53103be5 feat: add chat integration presets for api keys
```

## 升级前检查

先确认本地状态，避免把临时文件混入同步：

```bash
cd /Users/zhaolin/Documents/Lin/app/sub2api
git status --short
git branch --show-current
git log -1 --oneline
git remote -v
```

本次升级前，工作区有部署文档改动，因此先单独提交：

```bash
git add docs/SERVER_UPDATE_WORKFLOW.md
git commit -m "docs: prefer local binary server updates"
```

未跟踪的 `generated-images/` 和 `generated-videos/` 是本地生成内容，不纳入提交。

## 拉取官方更新

添加官方 remote：

```bash
git remote add upstream https://github.com/Wei-Shaw/sub2api.git 2>/dev/null || true
git fetch upstream --tags --prune
```

确认官方标签：

```bash
git ls-remote --tags https://github.com/Wei-Shaw/sub2api.git | rg '0\.1\.138'
```

本次官方 `v0.1.138` 标签存在，但标签内 `backend/cmd/server/VERSION` 仍是 `0.1.137`。官方 `main` 后续有 `chore: sync VERSION to 0.1.138 [skip ci]`，所以本次先合并标签，再合并官方 `main` 的后续小提交。

```bash
git merge --no-ff v0.1.138 -m "merge upstream v0.1.138"
git merge --no-ff upstream/main -m "merge upstream main after v0.1.138"
```

## 冲突处理

本次冲突文件只有：

- `.dockerignore`
- `Dockerfile`

处理原则：

- 保留官方对 `docs/legal/*.md` build-time import 的说明。
- 保留 `docs/legal/` 不被 `.dockerignore` 排除。
- `Dockerfile` 中保留 `COPY docs/legal/ /app/docs/legal/`，保证前端构建时法律协议 Markdown 可以被 Vite raw import。

处理后执行：

```bash
git add .dockerignore Dockerfile
git commit --no-edit
```

## 测试适配

官方更新后，`SettingsView.vue` 新增了从 `route.query.tab` 读取设置页 tab 的逻辑。原测试没有 mock `vue-router`，导致 `SettingsView.spec.ts` 全部因 `route.query` 为空对象缺失而失败。

本次补充：

- mock `useRoute`
- mock `useRouter().replace`
- stub `RouterLink`
- mock `extractI18nErrorMessage`

提交：

```bash
git add frontend/src/views/admin/__tests__/SettingsView.spec.ts
git commit -m "test: mock settings route in admin settings specs"
```

## 本地验证

前端依赖：

```bash
pnpm --dir frontend install --frozen-lockfile
```

前端类型检查：

```bash
pnpm --dir frontend typecheck
```

关键前端测试：

```bash
pnpm --dir frontend test:run \
  src/utils/__tests__/chatIntegrations.spec.ts \
  src/views/admin/__tests__/SettingsView.spec.ts \
  src/stores/__tests__/auth.spec.ts
```

关键后端测试：

```bash
cd /Users/zhaolin/Documents/Lin/app/sub2api/backend
go test ./internal/service -run 'Test.*(Image|Phone|Lottery|Chat|Quota|OpenAI)'
go test ./internal/handler -run 'Test.*(Image|Phone|Lottery|OpenAI|Gateway)'
```

生产前端构建：

```bash
cd /Users/zhaolin/Documents/Lin/app/sub2api
pnpm --dir frontend run build
```

Go embed 编译验证：

```bash
cd /Users/zhaolin/Documents/Lin/app/sub2api/backend
GOTOOLCHAIN=auto CGO_ENABLED=0 go build -tags embed -trimpath -o /tmp/sub2api-verify ./cmd/server
```

本次验证结果：

- `pnpm --dir frontend typecheck` 通过
- 关键前端测试 `46 passed`
- 后端 service 关键测试通过
- 后端 handler 关键测试通过
- 前端生产构建通过
- Go `-tags embed` 编译通过

## 推送 fork

```bash
git push origin main
```

推送后确认：

```bash
git log --oneline --decorate --max-count=6
git status --short
```

## 服务器部署

本次按 [SERVER_UPDATE_WORKFLOW.md](./SERVER_UPDATE_WORKFLOW.md) 中推荐的“本地构建二进制上传”方式部署。

### 1. 创建回滚镜像

```bash
ssh root@103.79.184.134 '
set -e
ROLLBACK_TAG="sub2api-local:rollback-$(date +%Y%m%d-%H%M%S)"
docker tag sub2api-local:latest "$ROLLBACK_TAG"
echo "$ROLLBACK_TAG" >/opt/sub2api-deploy/last-rollback-tag.txt
echo "rollback_tag=$ROLLBACK_TAG"
docker inspect -f "current_image={{.Image}} health={{.State.Health.Status}}" sub2api
'
```

本次回滚标签：

```text
sub2api-local:rollback-20260623-064712
```

### 2. 本地构建 Linux 二进制

```bash
cd /Users/zhaolin/Documents/Lin/app/sub2api

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
```

本次构建结果：

```text
version=0.1.138 commit=b54b794b
```

### 3. 打包资源和轻量 Dockerfile

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

### 4. 同步源码和产物

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

ssh root@103.79.184.134 'rm -rf /opt/sub2api-artifact && mkdir -p /opt/sub2api-artifact'
rsync -az --delete /tmp/sub2api-deploy-artifact/ root@103.79.184.134:/opt/sub2api-artifact/
```

### 5. 服务器重建应用镜像并重启

```bash
ssh root@103.79.184.134 '
set -e
cd /opt/sub2api-artifact
docker build -t sub2api-local:latest .

cd /opt/sub2api-deploy
docker compose -f docker-compose.local.yml up -d --no-deps --force-recreate sub2api
'
```

只重启 `sub2api`，不重启 PostgreSQL 和 Redis。

## 服务器回测

健康检查：

```bash
ssh root@103.79.184.134 '
docker inspect -f "{{.State.Health.Status}} {{.Image}}" sub2api
curl -fsS http://127.0.0.1:8088/health
cd /opt/sub2api-deploy
docker compose -f docker-compose.local.yml ps
'
```

版本确认：

```bash
ssh root@103.79.184.134 '/opt/sub2api-artifact/sub2api --version 2>/dev/null || true'
```

本次结果：

```text
Sub2API 0.1.138 (commit: b54b794b, built: 2026-06-23T06:47:32Z)
```

公网接口：

```bash
curl -I https://ai.lixnus.cc/
curl -sS https://ai.lixnus.cc/api/v1/settings/public | jq '.data | {site_name, registration_enabled, phone_verify_enabled, hide_ccs_import_button}'
```

本次结果：

```json
{
  "site_name": "ZemraAI",
  "registration_enabled": true,
  "phone_verify_enabled": true,
  "hide_ccs_import_button": false
}
```

生图分组模型列表轻测：

```bash
curl -sS --max-time 30 https://ai.lixnus.cc/v1/models \
  -H "Authorization: Bearer $API_KEY" \
  | jq '{count: (.data|length), image_models: [.data[]?.id | select(test("^gpt-image"))][0:5]}'
```

本次结果：

```json
{
  "count": 17,
  "image_models": [
    "gpt-image-1",
    "gpt-image-1.5",
    "gpt-image-2"
  ]
}
```

## 后续升级复用清单

1. 提交或暂存本地业务改动。
2. `git fetch upstream --tags --prune`。
3. 优先合并官方稳定 tag；如果 tag 后还有版本号同步提交，再合并官方 `main` 的必要小提交。
4. 解决冲突时优先保留本 fork 的业务功能，再吸收官方修复。
5. 跑前端 typecheck、关键前后端测试、生产构建、Go embed 编译。
6. 推送 `origin/main`。
7. 服务器先打 rollback tag。
8. 按本地二进制上传方式部署。
9. 回测健康检查、版本、首页、公共设置、关键业务接口。

## 注意事项

- 不要把 `generated-images/`、`generated-videos/`、`.env`、密钥、数据库备份提交到 Git。
- 生产部署不要执行 `docker compose down`，避免影响 PostgreSQL/Redis。
- 前端改动也必须重新构建 Go 二进制，因为前端资源通过 `-tags embed` 嵌入后端。
- 官方 tag 不一定同步 `VERSION` 文件，升级时要确认 `backend/cmd/server/VERSION`。
- 测试如果因官方新增路由/全局组件而失败，优先补测试 mock，不要绕过真实业务逻辑。
