#!/bin/bash

echo "=> 正在停止运行中的程序..."
pkill -f "auto-wallpaper" || true

echo "=> 正在移除二进制文件与源文件..."
rm -f ~/.local/bin/auto-wallpaper
rm -rf ~/.local/src/auto-wallpaper-go

echo "=> 正在删除开机自启项..."
rm -f ~/.config/autostart/auto-wallpaper.desktop

read -p "是否删除已下载的所有历史壁纸和 API 配置文件? (y/n): " confirm
if [ "$confirm" == "y" ]; then
    echo "=> 正在清理壁纸和配置目录..."
    rm -rf ~/.local/share/auto-wallpaper
    rm -rf ~/.config/auto-wallpaper
fi

echo "✅ 卸载完成。"
