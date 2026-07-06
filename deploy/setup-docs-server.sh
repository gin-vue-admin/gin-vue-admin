#!/usr/bin/env bash
# gva 文档站云主机初始化：创建专用部署账号 + SSH 通道。
# 仅部署 docs（VitePress 静态产物），不涉及 server/web。
#
# 用法（root）：
#   sudo bash deploy/setup-docs-server.sh
# 或覆盖默认值：
#   sudo DOCS_PATH=/www/wwwroot/your-domain bash deploy/setup-docs-server.sh
#
# 跑完后把打印的 Secrets/Variables 填进 GitHub，push 即触发 .github/workflows/deploy-docs.yml。
set -euo pipefail

DEPLOY_USER="${DEPLOY_USER:-gva-deploy}"
WWW_GROUP="${WWW_GROUP:-www}"
DOCS_PATH="${DOCS_PATH:-/www/wwwroot/gin-vue-admin.cncf.vip}"
KEY_NAME="gh_actions_gva"

[ "$(id -u)" -eq 0 ] || { echo "请用 root 执行： sudo bash $0" >&2; exit 1; }

# 1. 建专用部署账号并加入 www 组（最小权限，非 root）
id "$DEPLOY_USER" >/dev/null 2>&1 || useradd -m -s /bin/bash "$DEPLOY_USER"
id -nG "$DEPLOY_USER" | tr ' ' '\n' | grep -qxF "$WWW_GROUP" || usermod -aG "$WWW_GROUP" "$DEPLOY_USER"

# 2. 文档目录：不存在则创建；开组写 + setgid（新文件继承 www 组，便于 nginx/面板读写）
[ -d "$DOCS_PATH" ] || install -d -m 2775 -o "$DEPLOY_USER" -g "$WWW_GROUP" "$DOCS_PATH"
chmod -R g+w "$DOCS_PATH"
find "$DOCS_PATH" -type d -exec chmod g+s {} +

# 3. 生成专用 SSH 密钥（ed25519）并授权 authorized_keys
HOME_DIR="$(getent passwd "$DEPLOY_USER" | cut -d: -f6)"
install -d -m 700 -o "$DEPLOY_USER" -g "$WWW_GROUP" "$HOME_DIR/.ssh"
[ -f "$HOME_DIR/.ssh/$KEY_NAME" ] || sudo -u "$DEPLOY_USER" ssh-keygen -t ed25519 -C gh-actions-gva -f "$HOME_DIR/.ssh/$KEY_NAME" -N "" >/dev/null
sudo -u "$DEPLOY_USER" bash -c "cat ~/.ssh/$KEY_NAME.pub >> ~/.ssh/authorized_keys && chmod 600 ~/.ssh/authorized_keys"

# 4. 自测 SSH 闭环
if sudo -u "$DEPLOY_USER" ssh -i "$HOME_DIR/.ssh/$KEY_NAME" -o BatchMode=yes -o StrictHostKeyChecking=no "$DEPLOY_USER@127.0.0.1" 'echo deploy-ok' >/dev/null 2>&1; then
  echo "✓ SSH 自测通过"
else
  echo "⚠ SSH 自测失败，检查 sshd PubkeyAuthentication 与 ~/.ssh 权限（700/600）" >&2
fi

# 5. 打印 GitHub Secrets / Variables
PUB_IP="$(curl -s --max-time 3 https://api.ipify.org 2>/dev/null || echo '<服务器公网IP>')"
echo ""
echo "═══ 填入 GitHub Settings → Secrets and variables → Actions ═══"
echo "Secrets:"
echo "  SERVER_HOST    = $PUB_IP"
echo "  SERVER_USER    = $DEPLOY_USER"
echo "  DOCS_PATH      = $DOCS_PATH"
echo "  SERVER_SSH_KEY = （下面整段私钥，含 BEGIN/END 行）"
echo "──────────────────── 复制私钥 ───────────────────"
cat "$HOME_DIR/.ssh/$KEY_NAME"
echo "──────────────────────────────────────────────────"
echo "Variables:"
echo "  DOCS_BASE      = /"
echo ""
echo "✅ 配置后 git push origin main 即自动部署文档站"
