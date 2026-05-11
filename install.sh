#!/bin/bash
set -e

echo "=> 正在安装系统依赖..."
# 添加了 zenity 以支持弹出输入框
sudo pacman -S --needed go base-devel gtk3 libappindicator-gtk3 zenity

export GO111MODULE=on
export GOPROXY=https://goproxy.cn,direct

echo "=> 正在初始化并下载依赖 (GitHub 镜像加速)..."
go mod init auto-wallpaper 2>/dev/null || true
go get github.com/getlantern/systray

echo "=> 正在编译程序..."
go build -ldflags="-s -w" -o auto-wallpaper main.go

echo "=> 正在安装二进制文件到 ~/.local/bin..."
mkdir -p ~/.local/bin
cp auto-wallpaper ~/.local/bin/

echo "=> 正在创建开机自启项..."
mkdir -p ~/.config/autostart
cat << EOF > ~/.config/autostart/auto-wallpaper.desktop
[Desktop Entry]
Type=Application
Exec=$HOME/.local/bin/auto-wallpaper
Hidden=false
NoDisplay=false
X-GNOME-Autostart-enabled=true
Name=Auto ACG Wallpaper
Icon=preferences-desktop-wallpaper
EOF

echo "=> 启动程序..."
pkill -f "auto-wallpaper" || true
nohup ~/.local/bin/auto-wallpaper > /dev/null 2>&1 &
disown

echo "✅ 安装完成！请在顶栏查看新图标。"
