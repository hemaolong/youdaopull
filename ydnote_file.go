package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// 2021.11.26 有道云笔记格式
// {
//         "fileEntry":{
//             "checksum":null,
//             "createProduct":null,
//             "createTimeForSort":1628923523,
//             "deleted":false,
//             "dir":false,
//             "dirNum":0,
//             "domain":1,
//             "entryProps":{
//                 "bgImageId":"",
//                 "encrypted":"false",
//                 "modId":"",
//                 "noteSourceType":"0",
//                 "orgEditorType":"0"
//             },
//             "entryType":0,
//             "erased":false,
//             "favorited":false,
//             "fileNum":0,
//             "fileSize":1945,
//             "hasComment":false,
//             "id":"WEB74368e802e36521ddcc6ae03a37893df",
//             "modDeviceId":"",
//             "modifyTimeForSort":1629184068,
//             "myKeep":false,
//             "myKeepAuthor":"",
//             "myKeepAuthorV2":"",
//             "myKeepV2":false,
//             "name":"JPS-寻路.md",
//             "namePath":null,
//             "noteSourceType":0,
//             "noteTextSize":0,
//             "noteType":"0",
//             "orgEditorType":0,
//             "parentId":"6DCE03FC20BE4D35BBAFDBE6F8CEABA2",
//             "publicShared":false,
//             "rightOfControl":0,
//             "subTreeDirNum":0,
//             "subTreeFileNum":0,
//             "summary":"",
//             "tags":"",
//             "transactionId":"WEB74368e802e36521ddcc6ae03a37893df",
//             "transactionTime":1629184068,
//             "userId":"weixinobU7VjubR0sxrUa558x6tXrZp4X4",
//             "version":19250
//         },
//         "fileMeta":{
//             "author":null,
//             "chunkList":null,
//             "contentType":null,
//             "coopNoteVersion":0,
//             "createTimeForSort":1628923523,
//             "externalDownload":[

//             ],
//             "fileSize":1945,
//             "metaProps":{
//                 "FILE_IDENTIFIER":"A360A507DFBB477FA1119F070C918CE7",
//                 "WHOLE_FILE_TYPE":"NOS",
//                 "spaceused":"1945",
//                 "tp":"0"
//             },
//             "modifyTimeForSort":1629184069,
//             "resourceMime":null,
//             "resourceName":null,
//             "resources":[

//             ],
//             "sharedCount":0,
//             "sourceURL":"",
//             "storeAsWholeFile":true,
//             "title":"JPS-寻路.md"
//         },
//         "ocrHitInfo":[

//         ],
//         "otherProp":{

//         }
//     },

type YdResource struct {
	ResourceId      string
	ResourceType    int
	ResourceSubType int
	Version         int
}

// type YdEntries struct {
// 	Entries []*YdNoteFile
// }

type YdNoteFile struct {
	FileEntry struct {
		ID                string
		Name              string
		ParentID          string
		SubTreeDirNum     int
		SubTreeFileNum    int
		CreateTimeForSort int64
		Deleted           bool
		Dir               bool
		DirNum            int
		Tags              string
		UserID            string
	}

	FileMeta struct {
		FileSize          int64
		ModifyTimeForSort int64
		Resources         []YdResource
		SourceURL         string
		Title             string
	}

	Children []*YdNoteFile
}

func (yf *YdNoteFile) IsUpdated(f *YdNoteFile) bool {
	return f.Size() != yf.Size() || f.ModTime().Unix() != yf.ModTime().Unix()
}

// 收藏的笔记
func (yf *YdNoteFile) GetSourceURL() string {
	return yf.FileMeta.SourceURL
}

// 上传的笔记

func (yf *YdNoteFile) Dict() *zerolog.Event {
	return zerolog.Dict().Str("name", yf.Name()).Str("id", yf.ID()).
		Int64("size", yf.Size())
}

// fs.File
// 名字不是唯一的，以id为准
func (yf *YdNoteFile) ID() string {
	return yf.FileEntry.ID
}

func (yf *YdNoteFile) Name() string {
	return yf.FileEntry.Name
}

func (yf *YdNoteFile) NeedExport2Docx() bool {
	fileExtension := filepath.Ext(yf.FileEntry.Name)
	return fileExtension == ".note" || fileExtension == ".html" || fileExtension == ""
}

func (yf *YdNoteFile) GetExport2DocxName() string {
	fileExtension := filepath.Ext(yf.FileEntry.Name)

	return strings.TrimSpace(strings.TrimSuffix(yf.FileEntry.Name, fileExtension)) + ".docx"
}

func (yf *YdNoteFile) Size() int64 {
	return yf.FileMeta.FileSize
}

func (yf *YdNoteFile) ModTime() time.Time {
	return time.Unix(yf.FileMeta.ModifyTimeForSort, 0)
}
func (yf *YdNoteFile) IsDir() bool {
	return yf.FileEntry.Dir
}

// 本地文件缓存信息，用于增量拉取
type YdFileSystem struct {
	ydNoteSession

	files      map[string]*YdNoteFile // 所有文件，包含目录
	cacheFiles map[string]*YdNoteFile
}

func (yfs *YdFileSystem) UpdateFile(f *YdNoteFile) {
	yfs.files[f.ID()] = f
}

func (yfs *YdFileSystem) Init(ydContext *YDNoteContext) error {
	yfs.files = make(map[string]*YdNoteFile)
	yfs.cacheFiles = yfs.loadCache()

	doYoudaoNoteLogin(ydContext, entryURL, yfs.startPull)
	return nil
}

func (yfs *YdFileSystem) GetFileExportFullPath(id string) (string, error) {
	result := []string{}
	curID := id
	for {
		f, ok := yfs.files[curID]
		if !ok {
			break
		}
		if len(result) == 0 {
			result = append(result, trimFileName(f.Name()))
		} else {
			result = append(result, f.Name())
		}

		curID = f.FileEntry.ParentID
		if len(curID) == 0 {
			break
		}
	}

	reverseResult := make([]string, 0, len(result))
	for i := len(result) - 1; i >= 0; i-- {
		reverseResult = append(reverseResult, result[i])
	}

	if len(reverseResult) == 0 {
		return "", fmt.Errorf("fail to find file in system id:%s", id)
	}
	fullPathStr := localExpordWordDir(reverseResult...)
	fullPathStr = strings.TrimSuffix(fullPathStr, ".note")
	fullPathStr += ".docx"
	return filepath.Abs(fullPathStr)
}

// 遍历线上目录，拉去所有文件简要信息，重组本地文件系统
func (yfs *YdFileSystem) walkRemoteFile(ydContext *YDNoteContext, parentName, parentTitle string) {
	// 无论如何拉取远程根目录信息
	topLevelFiles, err := yfs.listDir(ydContext, parentName)
	if err != nil {
		log.Error().Str("parent_name", parentName).Str("parent_title", parentTitle).Err(err).Msg("list dir fail")
		return
	}

	for _, v := range topLevelFiles {
		yfs.UpdateFile(v)

		if v.IsDir() {
			yfs.walkRemoteFile(ydContext, v.ID(), v.Name())
		}
	}
}

func getFileLocalPath(files map[string]*YdNoteFile, f *YdNoteFile) string {
	tmp := make([]string, 0, 10)
	pf := f
	var ok bool
	for {
		// 共享给我的文档
		if pf.FileEntry.ParentID == "-2" {
			tmp = append(tmp, "_sharein_")
		}
		if pf, ok = files[pf.FileEntry.ParentID]; ok {
			tmp = append(tmp, pf.Name())
		} else {
			break
		}
	}
	if len(tmp) >= 2 {
		for i := 0; i < len(tmp)/2; i++ {
			tmp[i], tmp[len(tmp)-i-1] = tmp[len(tmp)-i-1], tmp[i]
		}
	}
	tmp = append(tmp, f.Name())

	return localFileDir(tmp...)
}

// 增量拉取，只拉取修改日志或者文件内容发生变化的日志
func doDeltaPull(ctx context.Context, ydContext *YDNoteContext, cacheFiles, newFiles map[string]*YdNoteFile) error {
	refURLs := map[string]*YdNoteFile{}

	// 增加、更新操作
	for k, v := range newFiles {
		if v.IsDir() {
			continue
		}

		// if !strings.Contains(v.Name(), ".") {
		// 	log.Info().Str("name", v.Name()).Msg("----")
		// }

		// 收藏的链接，不直接下载
		if len(v.GetSourceURL()) > 0 {
			refURLs[v.ID()] = v
		}

		var skipReason string
		if v.FileMeta.FileSize > 0 {
			if old, ok := cacheFiles[k]; ok {
				if v.NeedExport2Docx() {
					fullPath, _ := ydContext.yfs.GetFileExportFullPath(v.ID())
					if _, inErr := os.Stat(fullPath); inErr == nil {
						skipReason = "exist(docx)"
					}
				} else if !old.IsUpdated(v) {
					fullPath := getFileLocalPath(newFiles, v)
					if _, inErr := os.Stat(fullPath); inErr == nil {
						skipReason = "exist(plain)"
					}
				}
			} else {
				if v.NeedExport2Docx() {
					fullPath, _ := ydContext.yfs.GetFileExportFullPath(v.ID())
					if _, inErr := os.Stat(fullPath); inErr == nil {
						skipReason = "exist(docx)"
					}
				}
			}
		} else {
			if v.FileEntry.ParentID == "-2" {
				skipReason = "回收站"
			} else if v.NeedExport2Docx() {
				skipReason = "empty(docx)"
			} else {
				skipReason = "empty"
			}
		}
		if len(skipReason) > 0 {
			log.Info().Dict("new", v.Dict()).Str("opt", skipReason).Msg("file skiped")
		} else {
			err := downloadFile(ctx, ydContext, v, getFileLocalPath(newFiles, v))
			log.Info().Err(err).Dict("new", v.Dict()).Str("opt", "+").Msg("file download")
		}
	}

	// 链接不直接下载，只输出链接列表？
	if len(refURLs) > 0 {
		data, _ := json.MarshalIndent(refURLs, "", " ")
		err := os.WriteFile(localCacheDir(refFileInfo), data, 0755)
		log.Info().Err(err).Int("file_count", len(refURLs)).Msg("save ref file info")
	}

	// 删除本地存在，远程不存在的文件
	for k, v := range cacheFiles {
		if v.IsDir() {
			continue
		}

		if _, ok := newFiles[k]; !ok {
			// os.Remove(getFileLocalPath(cacheFiles, v)) // 不再真正删除
			log.Info().Dict("old", v.Dict()).Str("opt", "-").Msg("file deleted")
		}
	}

	data, err := json.MarshalIndent(newFiles, "", " ")
	if err != nil {
		return err
	}
	return os.WriteFile(localCacheDir(localFileInfo), data, 0755)
}

func (yfs *YdFileSystem) startPull(ctx context.Context, ydContext *YDNoteContext) {
	defer ydContext.ContextCancel()

	// 无论如何拉取远程根目录信息
	begin := time.Now()
	log.Info().Msg("start pull remote file info")
	yfs.walkRemoteFile(ydContext, "/", "")         // 拉取我的笔记列表
	yfs.walkRemoteFile(ydContext, "_myshare_", "") // 拉取与我分享的笔记列表
	log.Info().Dur("cost", time.Since(begin)).Int("file_count", len(yfs.files)).Msg("pull remote file info finish")

	// 对比本地缓存，确定删除、更新、还是增加
	begin = time.Now()
	log.Info().Msg("start download remote file")
	cache := yfs.loadCache()
	ydContext.yfs = yfs
	err := doDeltaPull(ctx, ydContext, cache, yfs.files)

	// Wait all finish
	waitUntil(time.Now(), time.Second*300, ctx, func() bool {
		count := 0
		downloadFileAbsPath.Range(func(k, _ interface{}) bool {
			count++
			log.Info().Interface("file", k).Msg("wait file to download finish")
			return false
		})

		return count == 0
	})
	log.Info().Err(err).Dur("cost", time.Since(begin)).Int("file_count", len(yfs.files)).Msg("download remote file finish")
}

// 加载本地缓存文件
func (yfs *YdFileSystem) loadCache() map[string]*YdNoteFile {
	cf := localCacheDir(localFileInfo)
	if _, inErr := os.Stat(cf); os.IsNotExist(inErr) {
		return nil
	}
	data, err := os.ReadFile(cf)
	if err != nil {
		log.Error().Err(err).Msg("skip local cache file info")
		return nil
	}

	files := make(map[string]*YdNoteFile)
	err = json.Unmarshal(data, &files)
	if err != nil {
		log.Error().Err(err).Msg("skip local cache file info")
		return nil
	}
	log.Info().Int("file_count", len(files)).Msg("load cache file info")
	return files
}

// func (yfs *YdFileSystem) Open(name string) (fs.File, error) {
// 	if f, ok := yfs.files[name]; ok {
// 		return f, nil
// 	}
// 	return nil, fmt.Errorf("file not found:%s", name)
// }

// func (yfs *YdFileSystem) ReadDir(name string) ([]fs.DirEntry, error) {
// 	f, err := yfs.Open(name)
// 	if err != nil {
// 		return nil, err
// 	}
// 	if stat, _ := f.Stat(); !stat.IsDir() {
// 		return nil, fmt.Errorf("%s is not dir", name)
// 	}

// 	return nil, nil
// }

// func (yf *YdNoteFile) Mode() fs.FileMode {
// 	if yf.IsDir() {
// 		return fs.ModeDir
// 	}
// 	return fs.ModeDevice
// }

// func (yf *YdNoteFile) Sys() interface{} {
// 	return nil
// }

// func (yf *YdNoteFile) Stat() (fs.FileInfo, error) {
// 	return yf, nil
// }

// func (yf *YdNoteFile) Read([]byte) (int, error) {
// 	return 0, nil
// }
// func (yf *YdNoteFile) Close() error {
// 	return nil
// }

// fs.DirEntry
// func (yf *YdNoteFile) Type() fs.FileMode {
// 	return yf.Mode()
// }

// func (yf *YdNoteFile) Info() (fs.FileInfo, error) {
// 	return yf.Stat()
// }

// fs.ReadDir
// func (yf *YdNoteFile) ReadDir(n int) ([]fs.DirEntry, error) {
// 	result := make([]fs.DirEntry, 0, 30)
// 	if n < 0 {
// 		n = len(yf.Children)
// 	}
// 	for i := 0; i < n; i++ {
// 		result = append(result, yf.Children[i])
// 	}
// 	return result, nil
// }
