#!/bin/sh
# AnqiCMS 卸载脚本（Linux）
#
# 用法：
#   sudo sh uninstall.sh               交互式：停服务 → 删单元 → 询问是否删安装目录
#   sudo sh uninstall.sh /opt/anqicms  直接指定安装目录（不再询问，直接删除）
#   curl -fsSL <镜像>/uninstall.sh | sh
#                                      管道方式也能用：提示从 /dev/tty 读，
#                                      没有终端时只停服务、不删目录（防误删）
#
# 做的事：停并禁用 systemd 服务 → 删单元文件 → 询问是否删除安装目录
# 不会动数据库里的数据（库请按需自行 DROP DATABASE）。
#
# 可覆盖的环境变量：
#   ANQICMS_SERVICE  systemd 服务名（默认 anqicms，与 deploy/anqicms.service 一致）
#   ANQICMS_DIR      安装目录（默认取服务单元里的 WorkingDirectory）

set -eu

c_yellow='\033[33m'; c_off='\033[0m'
warn() { printf "${c_yellow}[!]${c_off} %s\n" "$*"; }
say()  { printf '%s\n' "$*"; }

if [ "$(id -u)" -ne 0 ]; then
  say "需要 root 权限，请用：sudo sh $0" >&2
  exit 1
fi

SERVICE="${ANQICMS_SERVICE:-anqicms}"
DIR="${1:-${ANQICMS_DIR:-}}"

# 交互输入走 fd 3：用 `curl ... | sh` 时 stdin 是脚本本身，直接 read 会把
# 脚本后面的内容吃掉；能打开 /dev/tty 就从终端读，打不开就当作非交互。
CAN_ASK=0
if exec 3</dev/tty 2>/dev/null; then CAN_ASK=1; fi

ask() { # ask <提示语> → 答案写到 ANSWER
  printf '%s' "$1"
  if [ "$CAN_ASK" -eq 1 ]; then
    IFS= read -r ANSWER <&3 || ANSWER=""
  else
    ANSWER=""
  fi
}

# 安装目录优先从服务单元里读（install.sh 写进去的是 WorkingDirectory），
# 这样不带参数也能删对目录，避免在 /root 下执行时误删当前目录。
if [ -z "$DIR" ] && command -v systemctl >/dev/null 2>&1; then
  DIR=$(systemctl show -p WorkingDirectory --value "$SERVICE" 2>/dev/null | head -n 1 || true)
  # 取不到（服务不存在）时 systemctl 会给 "/" 或空串，都不当作有效安装目录
  if [ "$DIR" = "/" ]; then DIR=""; fi
fi

# ---- 1. 停服务 ----
if command -v systemctl >/dev/null 2>&1; then
  systemctl stop "$SERVICE" 2>/dev/null || true
  systemctl disable "$SERVICE" 2>/dev/null || true
  if [ -f "/etc/systemd/system/$SERVICE.service" ]; then
    rm -f "/etc/systemd/system/$SERVICE.service"
    systemctl daemon-reload 2>/dev/null || true
    say "已移除 systemd 服务：$SERVICE"
  else
    say "未找到 systemd 单元：$SERVICE（可能不是用 systemd 启动的）"
  fi
else
  pkill -f "$SERVICE" 2>/dev/null || true
  say "已尝试结束 $SERVICE 进程"
fi

# ---- 2. 询问是否删安装目录 ----
if [ -z "$DIR" ]; then
  if [ "$CAN_ASK" -eq 1 ]; then
    ask "是否删除安装目录（含 config.json 与 public/uploads 上传文件）？[y/N]: "
    case "${ANSWER:-}" in
      y|Y|yes|YES)
        ask "安装目录 [$(pwd)]: "
        DIR="${ANSWER:-$(pwd)}"
        ;;
      *) DIR="" ;;
    esac
  else
    warn "没有可交互的终端，跳过删除安装目录（要删除请带参数：sh uninstall.sh <安装目录>）"
  fi
fi

if [ -n "$DIR" ]; then
  [ -d "$DIR" ] || { say "目录不存在：$DIR" >&2; exit 1; }
  say "删除 $DIR ..."
  rm -rf "$DIR"
  say "安装目录已删除。"
fi

warn "数据库中的数据没有删除，如需清理请手动执行 DROP DATABASE。"
warn "环境变量文件 /etc/anqicms.env 未动（里面可能有数据库密码），需要的话请手动删除。"
say "卸载完成。"
