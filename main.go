package main

import (
	"flag"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	flag.StringVar(&ydLocalDir, "yn_local_dir", "./ydnote", "本地日志存放根目录")
	flag.StringVar(&ydRemoteDir, "yn_remote_dir", "/", "有道云笔记日志根目录，只能配置第一层目录\n日志太多的朋友可以选择拉取目录")
	flag.Parse()

	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})

	ctx := CreateContext()

	ys, err := createYdNoteSession()
	if err != nil {
		log.Fatal().Err(err).Msg("启动失败")
	}

	ys.start(ctx)
}
