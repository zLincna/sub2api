# 手机号注册改造与部署说明

本文档记录本次手机号注册能力的代码调整、配置项、部署方式和回测步骤，后续从 fork 仓库更新服务器时可按此流程执行。

## 目标

- 注册仍然使用邮箱作为登录账号。
- 注册时新增手机号字段，并校验手机号唯一。
- 支持配置是否开启手机号短信验证码。
- 开启手机号短信验证码后，注册必须填写手机号和短信验证码。
- 短信验证码使用阿里云短信能力。
- 支持后续本地改代码、提交到 GitHub fork、服务器拉取并重新部署。

## 代码调整概要

### 后端

- 注册请求新增字段：
  - `phone`
  - `phone_verify_code`
- 新增接口：
  - `POST /api/v1/auth/send-phone-verify-code`
- 新增手机号归一化和校验逻辑：
  - 当前按中国大陆手机号处理。
  - 存储为 11 位数字。
- 新增手机号唯一性校验：
  - 注册前检查手机号是否已存在。
  - 数据库层增加唯一索引兜底。
- 新增阿里云短信发送与校验逻辑：
  - 发送验证码。
  - 校验验证码。
  - 支持通过数据库设置或环境变量读取配置。
- 邮箱验证注册流程兼容手机号：
  - 开启邮箱验证时，前端跳转到邮箱验证码页后仍会保留手机号和短信验证码。

### 数据库

新增迁移文件：

```text
backend/migrations/151_add_user_phone_registration.sql
```

新增字段：

```sql
ALTER TABLE users ADD COLUMN IF NOT EXISTS phone varchar(32) NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN IF NOT EXISTS phone_verified_at timestamptz NULL;
```

新增唯一索引：

```sql
CREATE UNIQUE INDEX IF NOT EXISTS users_phone_unique_active
    ON users(phone)
    WHERE deleted_at IS NULL AND phone <> '';
```

应用启动时会自动执行 `backend/migrations/*.sql`，迁移记录在 `schema_migrations` 表中。

### 前端

- 注册页新增手机号输入框。
- 开启 `phone_verify_enabled` 后显示短信验证码输入框和发送按钮。
- 公开设置接口新增 `phone_verify_enabled`，用于控制注册页展示。
- 管理后台设置接口类型补充阿里云短信相关字段。

### Docker 构建

本次同时保留了 Docker 构建修复：

- `.dockerignore` 允许 `docs/legal/*.md` 进入构建上下文。
- `Dockerfile` 在前端构建前复制 `docs/legal/`，避免前端 raw markdown import 缺文件。

## 配置项

### 公开注册配置

| 配置项 | 说明 |
| --- | --- |
| `registration_enabled` | 是否开放注册 |
| `email_verify_enabled` | 是否开启邮箱验证码 |
| `phone_verify_enabled` | 是否开启手机号短信验证码 |

### 阿里云短信配置

数据库设置项和环境变量均可使用。数据库设置优先，环境变量作为 fallback。

| 环境变量 | 说明 |
| --- | --- |
| `ALIYUN_ACCESS_KEY_ID` / `ALIBABA_CLOUD_ACCESS_KEY_ID` | 阿里云 AccessKey ID |
| `ALIYUN_ACCESS_KEY_SECRET` / `ALIBABA_CLOUD_ACCESS_KEY_SECRET` | 阿里云 AccessKey Secret |
| `ALIYUN_SMS_SIGN_NAME` | 短信签名 |
| `ALIYUN_SMS_TEMPLATE_CODE` | 短信模板 Code |
| `ALIYUN_SMS_TEMPLATE_PARAM_KEY` | 模板验证码参数名，默认 `code` |
| `ALIYUN_SMS_TEMPLATE_STATIC_PARAMS` | 模板固定参数 JSON，默认 `{}` |
| `ALIYUN_SMS_SCHEME_NAME` | 阿里云短信 SchemeName，按阿里云控制台配置填写 |
| `ALIYUN_SMS_VALID_TIME_SECONDS` | 验证码有效期，默认 `300` |
| `ALIYUN_SMS_INTERVAL_SECONDS` | 重发间隔，默认 `60` |

## 推荐 Git 部署流程

后续推荐走 Git，而不是手工上传文件到服务器。

### 1. 本地提交并推送

```bash
git status --short
git add .
git commit -m "feat: add phone registration verification"
git push origin main
```

提交前确认不要把以下内容提交到仓库：

- 服务器密码
- API Key
- `.env`
- 数据库备份
- 临时压缩包
- 构建日志

### 2. 服务器拉取代码

服务器源码目录：

```bash
cd /opt/sub2api-src
git pull origin main
```

如果服务器之前不是通过 Git clone 得到的源码目录，可以重新 clone：

```bash
mv /opt/sub2api-src /opt/sub2api-src.bak.$(date +%Y%m%d%H%M%S)
git clone https://github.com/zLincna/sub2api.git /opt/sub2api-src
```

### 3. 服务器构建镜像

```bash
cd /opt/sub2api-src
docker build -t sub2api-local:latest .
```

如需保存日志：

```bash
cd /opt/sub2api-src
docker build -t sub2api-local:latest . > /opt/sub2api-deploy/build-phone.log 2>&1
tail -120 /opt/sub2api-deploy/build-phone.log
```

### 4. 重启服务

部署目录：

```bash
cd /opt/sub2api-deploy
docker compose -f docker-compose.local.yml up -d
```

确认 `docker-compose.local.yml` 中应用镜像使用的是本地构建镜像：

```yaml
services:
  sub2api:
    image: sub2api-local:latest
```

### 5. 查看日志

```bash
cd /opt/sub2api-deploy
docker compose -f docker-compose.local.yml logs --tail=120 sub2api
```

重点检查：

- 服务是否正常启动。
- 数据库迁移是否执行成功。
- 是否有 `151_add_user_phone_registration.sql` 相关错误。

## 临时文件同步方式

如果代码还没有提交到 GitHub，服务器无法通过 `git pull` 获得本地未提交修改。

这种情况下可以临时使用 `rsync`、`scp` 或压缩包同步到服务器，仅用于测试构建：

```bash
tar --exclude=.git --exclude=frontend/node_modules -czf /tmp/sub2api-phone.tgz .
scp /tmp/sub2api-phone.tgz root@SERVER_IP:/tmp/
ssh root@SERVER_IP 'tar -xzf /tmp/sub2api-phone.tgz -C /opt/sub2api-src'
```

长期发布仍建议使用 Git 提交和服务器拉取，方便回滚、审计和多人协作。

## 回测步骤

### 健康检查

```bash
curl -fsS http://127.0.0.1:8088/health
curl -fsS https://ai.lixnus.cc/api/v1/settings/public
```

### 注册接口基础校验

未填写手机号时应返回失败：

```bash
curl -i https://ai.lixnus.cc/api/v1/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"email":"test@example.com","password":"test123456"}'
```

手机号格式错误时应返回失败：

```bash
curl -i https://ai.lixnus.cc/api/v1/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"email":"test@example.com","phone":"123","password":"test123456"}'
```

### 短信验证码接口

开启 `phone_verify_enabled` 后，发送短信验证码：

```bash
curl -i https://ai.lixnus.cc/api/v1/auth/send-phone-verify-code \
  -H 'Content-Type: application/json' \
  -d '{"phone":"13800138000"}'
```

如果阿里云短信未配置，应返回类似：

```text
ALIYUN_SMS_NOT_CONFIGURED
```

配置完整后，应返回发送成功和倒计时。

### 注册成功链路

开启手机号短信验证码后，完整注册请求需要包含：

```json
{
  "email": "test@example.com",
  "phone": "13800138000",
  "phone_verify_code": "123456",
  "password": "test123456"
}
```

如果同时开启邮箱验证码，还需要带：

```json
{
  "verify_code": "邮箱验证码"
}
```

## 注意事项

- 当前短信验证码 challenge 存在应用进程内存中，单实例 Docker 部署可用。
- 如果未来做多实例负载均衡，短信 challenge 应迁移到 Redis，否则验证码可能发到 A 实例、注册请求落到 B 实例导致校验失败。
- 手机号唯一性同时由业务层和数据库唯一索引保证。
- 迁移文件一旦上线执行，不要修改旧迁移内容；如需调整，新增下一号迁移文件。
- 服务器部署建议保留数据库和 `.env` 备份后再升级。

## 常用排查命令

```bash
cd /opt/sub2api-deploy
docker compose -f docker-compose.local.yml ps
docker compose -f docker-compose.local.yml logs --tail=120 sub2api
docker compose -f docker-compose.local.yml logs --tail=120 postgres
```

检查迁移记录：

```bash
cd /opt/sub2api-deploy
docker compose -f docker-compose.local.yml exec postgres \
  psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" \
  -c "SELECT filename, applied_at FROM schema_migrations ORDER BY applied_at DESC LIMIT 10;"
```

检查用户表手机号字段：

```bash
cd /opt/sub2api-deploy
docker compose -f docker-compose.local.yml exec postgres \
  psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" \
  -c "\d users"
```
