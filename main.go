package main

import (
	"flag"
	"os"
	"runtime"

	"github.com/mattn/go-colorable"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	flag.StringVar(&ydLocalDir, "yn_local_dir", "./ydnote", "本地日志存放根目录")
	flag.StringVar(&ydLoginMode, "yn_login_mode", "wx", "登陆模式， wx/other, 默认是wx（微信）,否则弹出登陆页面")

	// flag.StringVar(&ydRemoteDir, "yn_remote_dir", "/", "有道云笔记日志根目录，只能配置第一层目录\n日志太多的朋友可以选择拉取目录")
	flag.BoolVar(&ydHeadless, "yn_headless", true, "是否打开chrome浏览器，如果二维码在控制台显示不正确请尝试打开浏览器，通过浏览器扫描二维码")
	flag.Parse()

	if runtime.GOOS == "windows" {
		terminalWriter = colorable.NewColorableStdout()
	} else {
		terminalWriter = os.Stdout
	}

	ydLoginMode = "other"
	if !isWXLogin() && ydHeadless {
		ydHeadless = false
		log.Warn().Msg("非微信登陆，必须打开浏览器")
	}

	log.Logger = log.Output(zerolog.ConsoleWriter{Out: terminalWriter})

	ctx := CreateContext()

	yfs := YdFileSystem{}
	err := yfs.Init(ctx)
	if err != nil {
		log.Fatal().Err(err).Msg("启动失败")
	}
}
