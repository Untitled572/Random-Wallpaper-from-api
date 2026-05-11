package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/getlantern/systray"
)

const (
	defaultAPI   = "https://api.yppp.net/pc.php"
	maxKeep      = 3
	minWidth     = 1920
	minHeight    = 1080
	maxRetries   = 3
	intervalMins = 30
)

var (
	saveDir    string
	configFile string
	currentAPI string
)

// 新图标：一个绿色的风景画框标识
const iconBase64 = `iVBORw0KGgoAAAANSUhEUgAAACAAAAAgCAYAAABzenr0AAAABHNCSVQICAgIfAhkiAAAAAlwSFlzAAALEwAACxMBAJqcGAAAAZ1JREFUWIXtl71Lw0AUxU9QHJwVB8HBRcVBxcXBwUVw0H/BRRz8H1z8w8HNf0BEHJwURwVBQUEQxMFVcBC1gzgo1TsfNCWJ+tqkVnrgQshzuffePbdJk6IoiqIoiqL8G2YAF8AFkIq5VwXQBbD1l7G51ARwA2AAyAPIAe0Y+2MANwBCP6Dq94D8m1K27vck1kYAawA2AYz1xZpT11oBlmKMRQAfAK58t2cArAMoA5j2xRpT1yoD2IsxNgfgyXd72U0oZ2gC+ABwBmDJF2tRXXsA4C3G2AqAhm7/A1hxD/z28tM+x9eG7tYBLLq9I0N7O4D1kDG73b22oY0NYBfAnNt71o1hE0C/G8MmgH43hk0A/W4M+wHsu71H3Rj2Ahhyey1DezeAJbdXNrS3B2Dc7S34HKvrmhHAoNvLB1g/A1hxexU/w0Bdz4gAXtxeL8D6NYAZt1cxDNR12Qhg3O3lA6y/AFhwexVDT11PiQAO3F4hwPpXgBlXwzCgLmpGADturxRg/RPAjKthGFAXRVEURVEU5d/5Blw1y6r2Q208AAAAAElFTkSuQmCC`

func init() {
	home, _ := os.UserHomeDir()
	
	// 壁纸保存目录
	saveDir = filepath.Join(home, ".local", "share", "auto-wallpaper")
	os.MkdirAll(saveDir, 0755)

	// 配置文件目录
	configDir := filepath.Join(home, ".config", "auto-wallpaper")
	os.MkdirAll(configDir, 0755)
	configFile = filepath.Join(configDir, "api.txt")

	// 读取存储的 API，如果没有则使用默认值
	b, err := os.ReadFile(configFile)
	if err == nil && len(bytes.TrimSpace(b)) > 0 {
		currentAPI = string(bytes.TrimSpace(b))
	} else {
		currentAPI = defaultAPI
	}
}

func main() {
	systray.Run(onReady, onExit)
}

func onReady() {
	icon, _ := base64.StdEncoding.DecodeString(iconBase64)
	systray.SetIcon(icon)
	systray.SetTitle("ACG Wallpaper")
	systray.SetTooltip("自动壁纸切换器")

	mNext := systray.AddMenuItem("⏭️ 立即切换", "拉取新壁纸")
	mChangeAPI := systray.AddMenuItem("⚙️ 更换 API", "修改壁纸来源")
	mOpen := systray.AddMenuItem("📂 打开图库", "查看已保存的壁纸")
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("❌ 退出程序", "关闭应用")

	go changeWallpaper()
	ticker := time.NewTicker(intervalMins * time.Minute)

	go func() {
		for {
			select {
			case <-mNext.ClickedCh:
				go changeWallpaper()
			case <-mChangeAPI.ClickedCh:
				go promptForAPI()
			case <-mOpen.ClickedCh:
				exec.Command("xdg-open", saveDir).Start()
			case <-mQuit.ClickedCh:
				systray.Quit()
			case <-ticker.C:
				go changeWallpaper()
			}
		}
	}()
}

func promptForAPI() {
	// 使用 zenity 呼出图形化输入框
	cmd := exec.Command("zenity", "--entry", "--title=更换 API 来源", "--text=请输入新的壁纸 API 链接\n(支持直接返回图片或 302 跳转的链接):", "--entry-text="+currentAPI)
	out, err := cmd.Output()
	
	if err == nil {
		newAPI := strings.TrimSpace(string(out))
		if newAPI != "" && newAPI != currentAPI {
			currentAPI = newAPI
			os.WriteFile(configFile, []byte(currentAPI), 0644)
			fmt.Println("API 已更新为:", currentAPI)
			// 修改 API 后立即拉取一张新壁纸测试
			go changeWallpaper()
		}
	}
}

func onExit() {}

func changeWallpaper() {
	for i := 0; i < maxRetries; i++ {
		err := processWallpaper()
		if err == nil {
			cleanupOldWallpapers()
			return
		}
		fmt.Printf("尝试 %d 失败: %v\n", i+1, err)
		time.Sleep(2 * time.Second)
	}
}

func processWallpaper() error {
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(currentAPI)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	config, format, err := image.DecodeConfig(bytes.NewReader(bodyBytes))
	if err != nil {
		return err
	}

	if config.Width < minWidth || config.Height < minHeight {
		return fmt.Errorf("尺寸不足: %dx%d", config.Width, config.Height)
	}

	filename := fmt.Sprintf("wallpaper_%d.%s", time.Now().Unix(), format)
	path := filepath.Join(saveDir, filename)
	os.WriteFile(path, bodyBytes, 0644)

	uri := "file://" + path
	exec.Command("gsettings", "set", "org.gnome.desktop.background", "picture-uri", uri).Run()
	exec.Command("gsettings", "set", "org.gnome.desktop.background", "picture-uri-dark", uri).Run()
	return nil
}

func cleanupOldWallpapers() {
	files, _ := filepath.Glob(filepath.Join(saveDir, "wallpaper_*"))
	if len(files) <= maxKeep {
		return
	}
	sort.Slice(files, func(i, j int) bool {
		fi, _ := os.Stat(files[i])
		fj, _ := os.Stat(files[j])
		return fi.ModTime().After(fj.ModTime())
	})
	for _, f := range files[maxKeep:] {
		os.Remove(f)
	}
}
