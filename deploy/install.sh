#!/usr/bin/env bash
#
# AnqiCMS 一键安装脚本（Linux）
# 用法：在解压后的目录中执行  sudo bash install.sh
#
# 做了什么：
#   1. 询问监听端口（默认 8001），写出 config.json
#   2. 给二进制加执行权限，创建运行时目录（data、logs、cache、public/uploads）
#   3. （可选）注册 systemd 服务并启动；否则用 nohup 后台启动
#   4. 提示到浏览器打开 /install 完成初始化（数据库、后台账号、演示数据）
#
# 与官方 Windows 版的区别是：数据库信息不在这里问，统一由 Web 安装向导
# （/install）填写——向导会建库建表、创建后台账号，也能识别已经导入过数据的库。
# 想跳过向导直接写好数据库，也可以设置环境变量后运行（见下）。
#
# 前置条件：
#   - 需要能访问的 MySQL 5.7+（库可以不存在，向导会自动建）
#   - 如用 systemd，请确保存在运行用户（默认 www，可在 deploy/anqicms.service 里改）
#
set -uo pipefail

cd "$(dirname "$0")" || exit 1

echo "=============================================="
echo "        AnqiCMS 一键安装 (Linux)"
echo "=============================================="

# ---- 1. 收集配置 ----
read -rp "监听端口 [默认 8001]: " PORT
PORT="${PORT:-8001}"

read -rp "站点名称 [默认 AnqiCMS]: " SITENAME
SITENAME="${SITENAME:-AnqiCMS}"

if [ ! -x ./anqicms ]; then
  chmod +x ./anqicms 2>/dev/null || true
fi

# ---- 2. 写出 config.json ----
# 已经存在就原样保留（升级场景：里面通常已经有 mysql 段与正式端口，
# 覆盖会把数据库连接丢掉）。
if [ -f config.json ]; then
  echo "已存在 config.json，保留原配置（不覆盖）"
else
  cat > config.json <<EOF
{
	"server": {
		"site_name": "$SITENAME",
		"env": "production",
		"port": $PORT,
		"log_level": "release"
	}
}
EOF
  chmod 600 config.json
  echo "已生成 config.json（端口 $PORT）"
fi

# ---- 3. 运行时目录 ----
mkdir -p data logs cache public/uploads public/static template system locales clientFiles
chmod -R u+rwX data logs cache public/uploads 2>/dev/null || true
echo "已创建运行时目录"

# ---- 4. 启动 ----
read -rp "是否注册为 systemd 服务并启动? [Y/n]: " USESYS
USESYS="${USESYS:-y}"
INSTALL_DIR="$(pwd)"
SERVICE_NAME="anqicms"

if [[ "$USESYS" =~ ^[Yy]$ ]] && command -v systemctl >/dev/null 2>&1; then
  RUN_USER="www"
  if ! id -u "$RUN_USER" >/dev/null 2>&1; then
    RUN_USER="root"
  fi
  sed -e "s#/www/wwwroot/anqicms#$INSTALL_DIR#g" \
      -e "s#^User=www#User=$RUN_USER#" \
      -e "s#^Group=www#Group=$RUN_USER#" \
      deploy/anqicms.service > "/etc/systemd/system/$SERVICE_NAME.service"
  if [ "$RUN_USER" != "root" ]; then
    chown -R "$RUN_USER":"$RUN_USER" "$INSTALL_DIR" 2>/dev/null || true
  fi
  systemctl daemon-reload
  systemctl enable "$SERVICE_NAME" >/dev/null 2>&1
  if systemctl restart "$SERVICE_NAME"; then
    echo "已通过 systemd 启动，查看状态： systemctl status $SERVICE_NAME"
  else
    echo "systemd 启动失败，请检查 /etc/systemd/system/$SERVICE_NAME.service（运行用户、路径）后执行："
    echo "  systemctl restart $SERVICE_NAME"
  fi
else
  nohup ./anqicms > logs/server.out.log 2>&1 &
  echo "已用 nohup 后台启动，日志： logs/server.out.log"
  sleep 2
fi

IPADDR="$(hostname -I 2>/dev/null | awk '{print $1}')"
[ -n "$IPADDR" ] || IPADDR="127.0.0.1"

echo "=============================================="
echo "安装完成（还差最后一步：初始化站点）"
echo "  打开： http://$IPADDR:$PORT/install"
echo "  在页面里填写 数据库地址/端口/库名/账号/密码、"
echo "  后台用户名密码、网站地址，勾选「演示数据」可带上示例内容。"
echo ""
echo "初始化时的数据会写进下面的目录："
echo "  程序目录： $INSTALL_DIR"
echo "  配置文件： $INSTALL_DIR/config.json"
echo "如需卸载： sudo sh uninstall.sh   （只删程序，不动数据库）"
echo "=============================================="
