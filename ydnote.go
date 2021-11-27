package main

import (
	"fmt"
	"net/url"
	"os"

	jsoniter "github.com/json-iterator/go"
	"github.com/rs/zerolog/log"
)

type ydNoteSession struct {
}

// 获取目录内文件列表
func (ys *ydNoteSession) listDir(ydContext *YDNoteContext, parent string) ([]*YdNoteFile, error) {
	var url string
	if parent == "/" {
		url = fmt.Sprintf(listEntireByParentPath, ydContext.Cstk, parent)
	} else {
		url = fmt.Sprintf(listEntireByParentID, parent, ydContext.Cstk)
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
		err = jsoniter.Unmarshal(respData, &result.Entries)
	} else {
		err = jsoniter.Unmarshal(respData, &result)
	}
	return result.Entries, err
}

// 获取目录内文件列表
func (ys *ydNoteSession) downloadFile(ydContext *YDNoteContext, f *YdNoteFile, localPath string) error {
	u, err := url.Parse(downLoadURL)
	if err != nil {
		return err
	}
	q := u.Query()
	q.Add("cstk", ydContext.Cstk)
	q.Add("fileId", f.Name())
	q.Add("version", "-1")
	q.Add("editorType", "-1")
	u.RawQuery = q.Encode()

	respData, err := HTTPReq(ydContext, "POST", u.String(),
		map[string]interface{}{
			"fileId":     f.Name(),
			"version":    -1,
			"convert":    "true",
			"editorType": 1,
			"cstk":       ydContext.Cstk,
		}, 60)
	if err != nil {
		return err
	}
	log.Info().Str("name", f.Name()).Str("title", f.FileMeta.Title).
		Int("file_size", len(respData)).
		Str("local_path", localPath).
		Msg("download file")

	return os.WriteFile(localPath, respData, 0755)
}

// // 获取有道云笔记指定文件夹 id，目前指定文件夹只能为顶层文件夹，如果要指定文件夹下面的文件夹，请自己改用递归实现
// func (ys *ydNoteSession) getDirID(ydContext *YDNoteContext, rootID, ydnoteDir string) (string, error) {
// 	respData, err := HTTPReqJSON(ydContext, "GET", fmt.Sprintf(DIR_MES_URL, rootID, ydnoteDir), nil, 10)
// 	if err != nil {
// 		return "", fmt.Errorf("有道路径解析错误(%w)", err)
// 	}

// 	entryID := respData.Get(fmt.Sprintf("entries.%s", ydnoteDir)).Str
// 	if len(entryID) == 0 {
// 		return "", fmt.Errorf("有道云笔记修改了接口地址，此脚本暂时不能使用！请提 issue2")
// 	}

// 	return entryID, nil
// }

// func (ys *ydNoteSession) pullDir(ydContext *YDNoteContext, parent string) {
// 	// 获取根目录所有文件
// 	rootID, err := ys.listDir(ydContext, parent)
// 	if err != nil {
// 		log.Error().Err(err).Msg("有道云笔记获取根目录文件夹错误")
// 		return
// 	}
// 	log.Info().Int("根目录文件数量", len(rootID)).Msg("开始下载笔记")

// }

// // 下载所有笔记
// func (ys *ydNoteSession) pullAll(ydContext *YDNoteContext, ydnoteDir, rootID string) error {
// 	// 检查本地路径，不存在则创建

// 	// 此处设置，后面会用，避免传参
// 	// ys.localDir = localDir

// 	// 检查有道路径
// 	if len(ydnoteDir) != 0 {
// 		rootID, err := ys.getDirID(ydContext, rootID, ydnoteDir)
// 		if err != nil {
// 			return err
// 		}

// 		log.Info().Str("dir", ydnoteDir).Str("root_id", rootID).Msg("")
// 		if len(rootID) == 0 {
// 			return fmt.Errorf("此文件夹「%s」不是顶层文件夹，暂不能下载！", ydnoteDir)
// 		}
// 	}

// 	return nil
// }

// // 递归遍历，根据 id 找到目录下的所有文件
// func (ys *ydNoteSession) walkDir(ydContext *YDNoteContext, id, localDir string) error {
// 	url := fmt.Sprintf(DIR_MES_URL, id, ydContext.Cstk)
// 	resp, err := HTTPReqJSON(ydContext, "GET", url, nil, 10)
// 	if err != nil {
// 		return fmt.Errorf("遍历子目录错误(%w)", err)
// 	}

// 	entries := resp.Get("entries").Array()
// 	if len(entries) == 0 {
// 		return fmt.Errorf("有道云笔记修改了接口地址，此脚本暂时不能使用！请提 issue3")
// 	}

// 	log.Info().Interface("entries", entries).Msg("xxx")
// 	// for _, subDir := range entries {
// 	// 	switch entries.Type {
// 	// 	case gjson.String: // 文件
// 	// 	case gjson.JSON: // 目录，继续遍历
// 	// 		for k, v := range subDir {
// 	// 			log.Info().Str("k", k).Interface("v", v).Msg("---------")
// 	// 		}

// 	// 	default:
// 	// 		return fmt.Errorf("有道云笔记修改了接口地址，此脚本暂时不能使用！请提 issue3")
// 	// 	}
// 	// }
// 	return nil

// }

// func (ys *ydNoteSession) start(ctx *YDNoteContext) {
// 	doYoudaoNoteLogin(ctx, WEB_URL, ys.startPull)
// }

// func createYdNoteSession() (*ydNoteSession, error) {
// 	dirs := []string{ydLocalDir, localFileDir(""), localCacheDir("")}
// 	for _, dir := range dirs {
// 		_, err := os.Stat(dir)
// 		if err != nil {
// 			err := os.MkdirAll(dir, 0755)
// 			if err != nil {
// 				return nil, fmt.Errorf("创建本地目录:%s失败(%w)，没有权限？", dir, err)
// 			}
// 		}
// 	}

// 	return &ydNoteSession{}, nil
// }
