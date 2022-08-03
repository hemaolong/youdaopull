package main

import (
	"context"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/rs/zerolog/log"
)

var (
	ydLocalDir  string // 本地文件目录
	ydRemoteDir string // 有道云笔记拉取的目录，只能是根目录
	ydHeadless  bool   // 是否打开chrome浏览器
	ydLoginMode string // 登陆模式，默认是扫码微信登陆

	// ydFileSystem = &YdFileSystem{}
	ydContext context.Context

	terminalWriter io.Writer

	sessionID string

	// downloadFileRelativePath sync.Map // 下载文件的相对目录，文件下载结束之后移动到相对目录
	downloadingFiles    sync.Map
	downloadFileAbsPath sync.Map
	downloadingCount    int32

	regpTrimFilename = regexp.MustCompile(`[\PP]+`)
)

const (
	apiListAllDir = `https://note.youdao.com/yws/open/notebook/all.json`
	entryURL      = "https://note.youdao.com/web/"
	// SIGN_IN_URL   = "https://note.youdao.com/signIn/index.html?&callback=https%3A%2F%2Fnote.youdao.com%2Fweb%2F&from=web" // 浏览器在传输链接的过程中是否都将符号转换为 Unicode？
	// LOGIN_URL = "https://note.youdao.com/login/acc/urs/verify/check?app=web&product=YNOTE&tp=urstoken&cf=6&fr=1&systemName=&deviceType=&ru=https%3A%2F%2Fnote.youdao.com%2FsignIn%2F%2FloginCallback.html&er=https%3A%2F%2Fnote.youdao.com%2FsignIn%2F%2FloginCallback.html&vcode=&systemName=&deviceType=&timestamp="

	// 列出所有book，根目录文件
	listEntireByParentPath = "https://note.youdao.com/yws/api/personal/file?method=listEntireByParentPath&keyfrom=web&path=%s"
	// 与我分享的笔记
	listEntriesRefNoteURL = `https://note.youdao.com/yws/api/personal/myshare/web?method=list&len=3000&sort=1&isReverse=false&keyfrom=web&sev=j1`

	listEntireByParentID = "https://note.youdao.com/yws/api/personal/file/%s?all=true&f=true&isReverse=false&keyfrom=web&len=3000&method=listPageByParentId&sort=1"

	// 笔记下载地址
	downloadNoteURL = "https://note.youdao.com/yws/api/personal/sync?method=download&keyfrom=web"
	// 资源下载地址
	downloadResURL = `https://note.youdao.com/yws/res/%d/%s`

	// downWorkURL = `https://note.youdao.com/ydoc/api/personal/doc?method=download-docx&fileId=%s&keyfrom=web`
	// word下载方式
	// def getNoteDocx(self, id, saveDir):
	//       url = 'https://note.youdao.com/ydoc/api/personal/doc?method=download-docx&fileId=%s&cstk=%s&keyfrom=web' % (id, self.cstk)
	//       response = self.get(url)
	//       with open('%s/%s.docx' % (saveDir, id), 'w') as fp:
	//           fp.write(response.content)
	// downloadWordURL = `https://hubble.netease.com/track/w/`
	// downloadWordURL = "https://note.youdao.com/ydoc/api/personal/doc?method=download-docx"
	setCurrentFileURL = `https://note.youdao.com/web/#/file/%s/note/%s/`
)

const (
	pngFile    = "qrcode.png"  // 二维码图片
	cookieFile = "cookies.tmp" // cookie

	localFileInfo = "file_info.json" // 本地文件信息，保留每个文件的版本信息避免每次都拉
	refFileInfo   = "ref_file_info.json"

	// loginSel = `body > ydoc-app > div > div:nth-child(1) > header > div > div > div.top-right > div.own-info > img`
	loginSel = `#flexible-left > div.sidebar.electron-drag > div.sidebar-header > app-personal > div > div.personal-info > img`
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

func localExpordWordDir(subDir ...string) string {
	tmp := make([]string, 0, 10)
	tmp = append(tmp, "__docx")
	tmp = append(tmp, subDir...)
	return localFileDir(tmp...)
}

func localDir(tmp ...string) string {
	ret := path.Join(tmp...)
	dir := path.Dir(ret)
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		log.Error().Err(err).Msg("mkdir fail")
	}
	return ret
}

func waitUntil(begin time.Time, duration time.Duration, ctx context.Context, cancelFunc func() bool) {
	var costTime time.Duration
	for {
		_ = chromedp.Sleep(time.Second).Do(ctx)
		costTime = time.Since(begin)

		if cancelFunc() || costTime >= duration {
			break
		}
	}
}

// 收藏笔记title名字会稍做调整
func trimFileName(t string) string {
	fileExtension := filepath.Ext(t)

	strs := regpTrimFilename.FindAllString(t, -1)
	ret := strings.Join(strs, "") + fileExtension
	return strings.ReplaceAll(ret, " ", "")
}
