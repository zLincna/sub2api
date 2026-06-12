# 服务器一键更新部署说明

本文档说明当前香港服务器上的 Sub2API 如何通过 Git 和 Docker 一键更新前后端。

当前服务器约定：

- 源码目录：`/opt/sub2api-src`
- 部署目录：`/opt/sub2api-deploy`
- Compose 文件：`/opt/sub2api-deploy/docker-compose.local.yml`
- 应用镜像：`sub2api-local:latest`
- 应用容器：`sub2api`
- 健康检查：`http://127.0.0.1:8088/health`

## 为什么需要脚本

这个项目的前端会在 Docker 构建时编译并嵌入 Go 后端二进制，所以前端或后端任一方有代码变化时，最终都需要重新构建应用镜像。

`deploy/update-server.sh` 把以下步骤合并成一次执行：

1. 拉取 GitHub 最新代码。
2. 判断是否有前端、后端、Dockerfile 或部署文件变化。
3. 有相关变化时构建 `sub2api-local:latest`。
4. 使用 `docker compose` 重启应用容器。
5. 等待健康检查。
6. 失败时尽量回滚到旧镜像。
7. 保存更新日志。

脚本不能完全消除编译时间，但可以避免每次手工输入命令和盯部署步骤。

## 首次放到服务器

如果服务器的 `/opt/sub2api-src` 已经拉到包含脚本的版本，可以执行：

```bash
install -m 755 /opt/sub2api-src/deploy/update-server.sh /usr/local/bin/sub2api-update
```

之后日常更新只需要：

```bash
sub2api-update
```

也可以不安装到 `/usr/local/bin`，直接运行：

```bash
bash /opt/sub2api-src/deploy/update-server.sh
```

## 日常更新

本地开发完成后先提交并推送：

```bash
git status --short
git add .
git commit -m "your change message"
git push origin main
```

服务器执行：

```bash
sub2api-update
```

如果不想等待终端输出，可以后台执行：

```bash
sub2api-update --detach
```

查看日志：

```bash
ls -lt /opt/sub2api-deploy/update-logs/
tail -f /opt/sub2api-deploy/update-logs/update-YYYYMMDD-HHMMSS.log
```

## 常用参数

强制重新构建，即使 Git 没有新提交：

```bash
sub2api-update --force
```

不拉 Git，只基于服务器当前源码构建：

```bash
sub2api-update --no-pull --force
```

只重启当前镜像，不构建：

```bash
sub2api-update --skip-build
```

后台强制构建：

```bash
sub2api-update --detach --force
```

## 环境变量覆盖

默认值已经适配当前服务器。如果目录或端口变更，可以临时覆盖：

```bash
SRC_DIR=/opt/sub2api-src \
DEPLOY_DIR=/opt/sub2api-deploy \
COMPOSE_FILE=docker-compose.local.yml \
IMAGE_NAME=sub2api-local:latest \
SERVICE_NAME=sub2api \
HEALTH_URL=http://127.0.0.1:8088/health \
sub2api-update
```

国内服务器拉 Go 依赖时可保留默认代理：

```bash
GOPROXY=https://goproxy.cn,direct GOSUMDB=sum.golang.google.cn sub2api-update
```

## 什么时候会跳过构建

如果远程没有新提交，脚本会跳过构建。需要重建时使用 `--force`。

如果有新提交，但变化不涉及以下路径，脚本也会跳过构建：

- `backend/`
- `frontend/`
- `docs/legal/`
- `Dockerfile`
- `.dockerignore`
- `deploy/docker-compose*`
- `deploy/docker-entrypoint.sh`

这种判断是为了避免只改文档时也花几分钟构建镜像。

## 回滚逻辑

脚本构建前会记录当前 `sub2api-local:latest` 的镜像 ID。

如果新容器启动后健康检查失败，脚本会尝试：

1. 把旧镜像重新标记为 `sub2api-local:latest`。
2. 重新 `docker compose up -d sub2api`。
3. 再次健康检查。

禁用自动回滚：

```bash
sub2api-update --no-rollback
```

## 手工排查命令

查看容器状态：

```bash
cd /opt/sub2api-deploy
docker compose -f docker-compose.local.yml ps
```

查看应用日志：

```bash
cd /opt/sub2api-deploy
docker compose -f docker-compose.local.yml logs --tail=120 sub2api
```

健康检查：

```bash
curl -fsS http://127.0.0.1:8088/health
curl -fsS https://ai.lixnus.cc/api/v1/settings/public
```

确认服务器代码版本：

```bash
cd /opt/sub2api-src
git log -1 --oneline
```

确认当前镜像：

```bash
docker images sub2api-local:latest
```

## 注意事项

- 不要在服务器源码目录保留未提交改动；脚本发现 dirty worktree 会直接停止。
- `.env`、数据库数据、备份文件和密钥不要提交到 Git。
- 生产更新建议优先使用 `sub2api-update --detach`，然后看日志。
- 如果 Docker build 第一次拉新 Go 依赖，会比平时慢；后续命中缓存会快很多。
