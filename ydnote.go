package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

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
func downloadFile(ydContext *YDNoteContext, f *YdNoteFile, localPath string) error {
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
	log.Info().Str("name", f.ID()).Str("name", f.Name()).
		Int("file_size", len(respData)).
		Str("local_path", localPath).
		Msg("download file")

	err = os.WriteFile(localPath, respData, 0755)
	if err != nil {
		return err
	}

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
	return nil
}
