#!/usr/bin/env bash
# 首次签发 Let's Encrypt 证书（standalone 模式）。
#
# 前置：
#   - .env 设 DOMAIN=your.domain.com 和 EMAIL=you@example.com
#   - 域名 A 记录已指向本机公网 IP
#   - 宿主 80 端口空闲（脚本会用 docker 跑 certbot standalone 临时占用 80）
#   - 已安装 docker
#
# 用法：bash deploy/scripts/init-certs.sh
set -eu

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
DEPLOY_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
ROOT_DIR="$(cd "$DEPLOY_DIR/.." && pwd)"

# 从 deploy/.env 读取
if [ -f "$DEPLOY_DIR/.env" ]; then
    # shellcheck disable=SC1091
    set -a; . "$DEPLOY_DIR/.env"; set +a
fi

: "${DOMAIN:?请在 deploy/.env 设 DOMAIN=your.domain.com}"
: "${EMAIL:?请在 deploy/.env 设 EMAIL=you@example.com}"

echo "[init-certs] 域名: $DOMAIN"
echo "[init-certs] 邮箱: $EMAIL"
echo "[init-certs] 证书输出: /etc/letsencrypt/live/$DOMAIN/"

# 1. 签发（standalone，临时占用宿主 80）
echo "[init-certs] 用 certbot standalone 签发（请确认宿主 80 空闲）..."
docker run --rm \
    -p 80:80 \
    -v /etc/letsencrypt:/etc/letsencrypt \
    -v /etc/letsencrypt-log:/var/log/letsencrypt \
    "${REGISTRY:-docker.io}/certbot/certbot:latest" \
    certonly --standalone \
    -d "$DOMAIN" -m "$EMAIL" \
    --agree-tos --no-eff-email

# 2. 生成 nginx 运行时配置（替换 DOMAIN_PLACEHOLDER）
echo "[init-certs] 生成 nginx.runtime.conf..."
mkdir -p "$DEPLOY_DIR/data/webroot"
sed "s|DOMAIN_PLACEHOLDER|$DOMAIN|g" "$DEPLOY_DIR/nginx.tls.conf" \
    > "$DEPLOY_DIR/data/nginx.runtime.conf"

# 3. 提示启动命令
cat <<EOF

[init-certs] 完成。证书路径：/etc/letsencrypt/live/$DOMAIN/

启动 TLS 全栈：
  cd "$DEPLOY_DIR"
  docker compose -f docker-compose.yml -f docker-compose.tls.yml --profile tls up -d

验证：
  curl -I https://$DOMAIN/api/health
  open https://$DOMAIN/docs/

续期由 certbot 服务自动处理（每 12h，仅临近过期才真正更新，更新后 HUP nginx reload）。
EOF
