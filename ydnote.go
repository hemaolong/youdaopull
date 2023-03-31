package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/chromedp"
	"github.com/rs/zerolog/log"
)

type ydNoteSession struct {
}

// 获取目录内文件列表
func (ys *ydNoteSession) listDir(ydContext *YDNoteContext, parent string) ([]*YdNoteFile, error) {
	var url string
	if parent == "/" {
		url = fmt.Sprintf(listEntireByParentPath, parent)
	} else if parent == "_myshare_" {
		url = listEntriesRefNoteURL
	} else {
		url = fmt.Sprintf(listEntireByParentID, parent)
	}
	respData, err := HTTPReq(ydContext, "GET", url, nil, 60)
	if err != nil {
		return nil, err
	}
	log.Info().Str("parent", parent).RawJSON("files", respData).Msg("list dir")

	result := struct {
		Entries []*YdNoteFile
	}{}

	// result := make([]*YdNoteFile, 0, 30)
	if respData[0] == '[' {
		err = json.Unmarshal(respData, &result.Entries)
	} else {
		err = json.Unmarshal(respData, &result)
	}

	// 为了可读性，本地用Title作为文件名，但是操作系统对文件名/目录名有特殊要求，需要转换
	for _, v := range result.Entries {
		v.FileMeta.Title, err = Filenamify(v.FileMeta.Title, Options{Replacement: "_", MaxLength: 200})
		if err != nil {
			log.Error().Err(err).Str("name", v.Name()).Msg("filenamify fail")
		}
	}

	return result.Entries, err
}

// 获取目录内文件列表
func downloadFile(ctx context.Context, ydContext *YDNoteContext, f *YdNoteFile, localPath string) error {
	u, err := url.Parse(downloadNoteURL)
	if err != nil {
		return err
	}
	q := u.Query()
	q.Add("fileId", f.ID())
	q.Add("version", "-1")
	q.Add("editorType", "-1")
	u.RawQuery = q.Encode()

	respData, err := HTTPReq(ydContext, "POST", u.String(),
		map[string]interface{}{
			"fileId":     f.ID(),
			"version":    -1,
			"convert":    "true",
			"editorType": 1,
		}, 60)
	if err != nil {
		return err
	}
	log.Info().Str("id", f.ID()).Str("name", f.Name()).
		Int("file_size", len(respData)).
		Str("local_path", localPath).
		Msg("download file")

	err = os.WriteFile(localPath, respData, 0755)
	if err != nil {
		return err
	}

	if f.NeedExport2Docx() {
		return exportToWord(ctx, ydContext, f, localPath)
	} else {
		// 下载笔记相关资源，例如图片
		if len(f.FileMeta.Resources) > 0 {
			ext := filepath.Ext(f.Name())
			resPath := strings.TrimSuffix(localPath, ext)
			resPath = resPath + "_res"
			err = os.MkdirAll(resPath, 0755)
			if err != nil {
				return err
			}
			for _, res := range f.FileMeta.Resources {
				resRemoteURL := fmt.Sprintf(downloadResURL, res.Version, res.ResourceId)
				resLocalPath := path.Join(resPath, res.ResourceId+".png")
				_, err = downloadImg(ydContext, resRemoteURL, resLocalPath)
				if err != nil {
					log.Error().Err(err).Dict("file", f.Dict()).Str("res_remote_url", resRemoteURL).Str("res_local_path", resLocalPath).Msg("download res fail")
					continue
				}
				log.Info().Err(err).Dict("file", f.Dict()).Str("res_remote_url", resRemoteURL).Str("res_local_path", resLocalPath).Msg("download res ok")
			}
		}
	}
	return nil
}

// downJS := `this.downloadService.download("/ydoc/api/personal/doc?method=download-docx&fileId=" + e.id), p.hubbleTracker.track(O.TrackerEvent.YnoteFunction, {
// 				fileFunction: O.TrackerActionType.ExportToWord
// 			})	`

// var executed *runtime.RemoteObject
// _ = chromedp.Evaluate(`
// 				console.log("hello world");
// 				console.log();
// 				`, executed).Do(ctx)

// {"dataType": "e","sessionUuid": "d85978d324aae49bbce2580054f031983ddc3827","userId": "3cec059660f2ea50e915f6dfb4c0d25c","currentUrl": "https://note.youdao.com/web/#/file/recent/note/8DE580E902054AD09E0FEEB5DF94035D/","referrer": "https://www.google.com.hk/","referrerDomain": "www.google.com.hk","sdkVersion": "1.6.12.8","sdkType": "js","deviceOs": "windows","deviceOsVersion": "Win10","devicePlatform": "web","browser": "chrome","browserVersion": "103.0.0.0","screenWidth": 1920,"screenHeight": 1080,"eventId": "ynoteFunction","appKey": "MA-B0D8-94CBE089C042","time": 1657799110876,"persistedTime": 1615443922703,"deviceUdid": "8ba5f2bc777a9ee2164b335cf8803df40ccda0e6","pageTitle": "有道云笔记","urlPath": "/web/","currentDomain": "note.youdao.com","pageOpenScene": "Browser","secondLevelSource": "www.google.com.hk","attributes": {"userId": "weixinobU7VjubR0sxrUa558x6tXrZp4X4","fileId": "8DE580E902054AD09E0FEEB5DF94035D","fileFunction": "exportWord","$_ntes_nnid_id": "f3762b731c85fda9950b4d8fbc95315d","$_ntes_nnid_time": "1646791100064","$_ntes_domain": "163.com","$P_INFO_userid": "13816366465","$P_INFO_time": "1657699292"}}
func exportToWord(ctx context.Context, ydContext *YDNoteContext, f *YdNoteFile, _ string) error {
	u, err := url.Parse(fmt.Sprintf(setCurrentFileURL, f.FileEntry.ParentID, f.ID()))
	if err != nil {
		return err
	}

	err = chromedp.Navigate(u.String()).Do(ctx)
	if err != nil {
		return err
	}

	// respData, err := HTTPReq(ydContext, "GET", u.String(),
	// 	nil, 60)
	// if err != nil {
	// 	return err
	// }

	// 等待显示成功
	exportFileName := f.GetExport2DocxName()
	exportFileNameWithoutSuffix := strings.TrimSuffix(exportFileName, ".docx")
	trimWant := trimFileName(exportFileNameWithoutSuffix)
	titleSel := `#hd-space-between > div > top-title > div > form > input`
	tryCount := 0
	waitUntil(time.Now(), time.Second*300, ctx, func() bool {
		var nodes []*cdp.Node
		if err = chromedp.Nodes(titleSel, &nodes, chromedp.NodeReady, chromedp.NodeEnabled).Do(ctx); err != nil {
			return false
		}
		if len(nodes) == 0 {
			return false
		}

		titleText := nodes[0].AttributeValue("title")
		trimGet := trimFileName(titleText)
		if trimWant == trimGet {
			tryCount = 0
			return true
		}

		// _, _ = HTTPReq(ydContext, "GET", u.String(),
		// 	nil, 60)
		tryCount++
		if tryCount >= 0 {
			_ = chromedp.Navigate(u.String()).Do(ctx)
			tryCount = 0
		}
		if exportFileNameWithoutSuffix != trimWant {
			log.Warn().Str("want", trimWant).
				Str("get", trimGet).Msg("wait to get file content(changed)")
		} else {
			log.Warn().Str("want", trimWant).
				Str("get", trimGet).Msg("wait to get file content")
		}
		return false
	})

	fullPathStr, err := ydContext.yfs.GetFileExportFullPath(f.ID())
	if err != nil {
		return err
	}

	log.Info().Str("id", f.ID()).Str("name", f.Name()).
		Str("local_path", fullPathStr).
		Msg("download file word")

	_ = chromedp.WaitVisible(`#flexible-left > div.sidebar.electron-drag > div.sidebar-header > app-personal > div > div`).Do(ctx)

	// toolbarSel := `#hd-space-between > toolbar > div > div:nth-child(3)`
	// toolbarSel := `#hd-space-between > div.title-right-container > toolbar > div > div:nth-child(3) > ul`
	toolbarSel := `#hd-space-between > div.title-right-container > toolbar > div > div:nth-child(3)`
	_ = chromedp.WaitVisible(toolbarSel).Do(ctx) // 打开菜单界面
	// _ = chromedp.WaitEnabled(toolbarSel).Do(ctx) //

	// 点出操作菜单
	err = chromedp.Click(toolbarSel).Do(ctx)
	if err != nil {
		return err
	}

	// 找到按钮
	// chromedp.Sleep(time.Microsecond * time.Duration(rand.Int31n(500)+1500)).Do(ctx)
	// exportAsWordSel := `#hd-space-between > toolbar > div > div:nth-child(3) > ul > li:nth-child(13)`
	exportAsWordSel := `#hd-space-between > div.title-right-container > toolbar > div > div:nth-child(3) > ul > li:nth-child(14) > span`
	_ = chromedp.WaitVisible(exportAsWordSel).Do(ctx)
	_ = chromedp.WaitEnabled(exportAsWordSel).Do(ctx)

	waitUntil(time.Now(), time.Second*300, ctx, func() bool {
		if atomic.LoadInt32(&downloadingCount) >= 12 {
			chromedp.Sleep(time.Second).Do(ctx)
			log.Info().Str("path", f.Name()).
				Msg("wait previous file to download finish")
			return false
		}
		return true
	})

	// 导出
	err = chromedp.Click(exportAsWordSel).Do(ctx)
	if err != nil {
		return err
	}
	atomic.AddInt32(&downloadingCount, 1)

	downloadFileAbsPath.Store(trimFileName(exportFileName), fullPathStr)

	// time.Sleep(time.Minute * 1115)

	// queryInputSel := `#file-list-search`
	// exportBtnSel := "#hd-space-between > toolbar > div > div:nth-child(3) > ul > li:nth-child(13)"

	// err := chromedp.Focus(queryInputSel).Do(ctx)
	// if err != nil {
	// 	return err
	// }

	// _ = chromedp.WaitVisible(exportBtnSel).Do(ctx)
	// // 利用input field传递变量
	// err = chromedp.SendKeys(queryInputSel, kb.End+f.ID()).Do(ctx)
	// if err != nil {
	// 	return err
	// }

	// var executed *runtime.RemoteObject
	// err = chromedp.Evaluate(`
	//       var inputField = document.querySelector("#file-list-search");
	// 			inputField.value = typeof this.launcher;
	// 			window.console.log("xxxx");
	// 			var btn = document.querySelector("#hd-space-between > toolbar > div > div:nth-child(3) > ul > li:nth-child(13)");
	// 			// 清除现有事件
	// 			btn.replaceWith(btn.cloneNode(true));
	// 			btn = document.querySelector("#hd-space-between > toolbar > div > div:nth-child(3) > ul > li:nth-child(13)");

	// 			btn.addEventListener('click', function handleClick(event) {
	//       	inputField.value =  typeof this.prototype; // typeof this.downloadAsWord;//this.parentView.parentView.context.explorer;
	// 				console.error(event.target);
	// 				console.error(window.onload);
	// 				const vars = Object.keys( window );
	// 				vars.forEach(function (value) {
	// 					console.error(value);
	// 				});

	// 				console.error("xxx");
	// 				// console.error(window);
	// 				// inputField.value = "xxxxxx";
	// 			});
	// 			`, executed).Do(ctx)
	// if err != nil {
	// 	fmt.Println(">>>", err)
	// 	os.Exit(0)
	// 	// return err
	// }

	// u, err := url.Parse(downloadWordURL)
	// if err != nil {
	// 	return err
	// }
	// q := u.Query()
	// q.Add("fileId", f.ID())
	// u.RawQuery = q.Encode()

	// // params := newExportParams(f)
	// // params["dataType"] = "e"
	// // if attrs, ok := params["attributes"].(map[string]interface{}); ok {
	// // 	attrs["fileFunction"] = "exportWord"
	// // }

	// // 顺便导出docx
	// fmt.Println("u.String()", u.String())
	// respData2, err := HTTPReq(ydContext, "GET", u.String(),
	// 	nil, 6000)
	// if err != nil {
	// 	return err
	// }
	// log.Info().Str("name", f.ID()).Str("name", f.Name()).
	// 	Str("file_size", string(respData2)).
	// 	Str("local_path", localPath).
	// 	Msg("download file")

	// err = os.WriteFile(localPath, respData2, 0755)
	// if err != nil {
	// 	return err
	// }
	return nil
}
