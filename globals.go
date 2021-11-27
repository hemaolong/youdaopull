package main

import (
	"context"
	"os"
	"path"

	"github.com/rs/zerolog/log"
)

var (
	ydLocalDir  string // 本地文件目录
	ydRemoteDir string // 有道云笔记拉取的目录，只能是根目录

	ydFileSystem = &YdFileSystem{}
	ydContext    context.Context
)

const (
	apiListAllDir = `https://note.youdao.com/yws/open/notebook/all.json`
	WEB_URL       = "https://note.youdao.com/web/"
	SIGN_IN_URL   = "https://note.youdao.com/signIn/index.html?&callback=https%3A%2F%2Fnote.youdao.com%2Fweb%2F&from=web" // 浏览器在传输链接的过程中是否都将符号转换为 Unicode？
	LOGIN_URL     = "https://note.youdao.com/login/acc/urs/verify/check?app=web&product=YNOTE&tp=urstoken&cf=6&fr=1&systemName=&deviceType=&ru=https%3A%2F%2Fnote.youdao.com%2FsignIn%2F%2FloginCallback.html&er=https%3A%2F%2Fnote.youdao.com%2FsignIn%2F%2FloginCallback.html&vcode=&systemName=&deviceType=&timestamp="

	// 列出所有book，根目录文件
	listEntireByParentPath = "https://note.youdao.com/yws/api/personal/file?method=listEntireByParentPath&keyfrom=web&cstk=%s&path=%s"
	listEntireByParentID   = "https://note.youdao.com/yws/api/personal/file/%s?all=true&cstk=%s&f=true&isReverse=false&keyfrom=web&len=3000&method=listPageByParentId&sort=1"

	DIR_MES_URL = "https://note.youdao.com/yws/api/personal/file/%s?all=true&f=true&len=200&sort=1&isReverse=false&method=listPageByParentId&keyfrom=web&cstk=%s"
	downLoadURL = "https://note.youdao.com/yws/api/personal/sync?method=download&keyfrom=web"
)

const (
	pngFile    = "qrcode.png"  // 二维码图片
	cookieFile = "cookies.tmp" // cookie

	localFileInfo = "file_info.json" // 本地文件信息，保留每个文件的版本信息避免每次都拉

	loginSel = `body > ydoc-app > div > div:nth-child(1) > header > div > div > div.top-right > div.own-info > img`
)

func localFileDir(subDir ...string) string {
	tmp := make([]string, 0, 10)
	tmp = append(tmp, ydLocalDir, "file")
	tmp = append(tmp, subDir...)
	return localDir(tmp...)
}

func localCacheDir(subDir ...string) string {
	tmp := make([]string, 0, 10)
	tmp = append(tmp, ydLocalDir, "cache")
	tmp = append(tmp, subDir...)
	return localDir(tmp...)
}

func localDir(tmp ...string) string {
	ret := path.Join(tmp...)
	dir := path.Dir(ret)
	err := os.MkdirAll(dir, 0o0755)
	if err != nil {
		log.Error().Err(err).Msg("mkdir fail")
	}
	return ret
}
