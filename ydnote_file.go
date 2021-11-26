package main

import (
	"fmt"
	"io/fs"
	"time"
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
		FizeSize          int64
		ModifyTimeForSort int64
		Resources         []string
		SourceURL         string
		Title             string
	}

	Children []*YdNoteFile
}

// fs.File
// 名字不是唯一的，以id为准
func (yf *YdNoteFile) Name() string {
	return yf.FileEntry.ID
}

func (yf *YdNoteFile) Size() int64 {
	return yf.FileMeta.FizeSize
}
func (yf *YdNoteFile) Mode() fs.FileMode {
	if yf.IsDir() {
		return fs.ModeDir
	}
	return fs.ModeDevice
}
func (yf *YdNoteFile) ModTime() time.Time {
	return time.Unix(yf.FileMeta.ModifyTimeForSort, 0)
}
func (yf *YdNoteFile) IsDir() bool {
	return yf.FileEntry.Dir
}

func (yf *YdNoteFile) Sys() interface{} {
	return nil
}

func (yf *YdNoteFile) Stat() (fs.FileInfo, error) {
	return yf, nil
}

func (yf *YdNoteFile) Read([]byte) (int, error) {
	return 0, nil
}
func (yf *YdNoteFile) Close() error {
	return nil
}

// fs.DirEntry
func (yf *YdNoteFile) Type() fs.FileMode {
	return yf.Mode()
}

func (yf *YdNoteFile) Info() (fs.FileInfo, error) {
	return yf.Stat()
}

// fs.ReadDir
func (yf *YdNoteFile) ReadDir(n int) ([]fs.DirEntry, error) {
	result := make([]fs.DirEntry, 0, 30)
	if n < 0 {
		n = len(yf.Children)
	}
	for i := 0; i < n; i++ {
		result = append(result, yf.Children[i])
	}
	return result, nil
}

// 本地文件缓存信息，用于增量拉取
type YdFileSystem struct {
	files map[string]*YdNoteFile // 所有文件，包含目录
}

func (yfs *YdFileSystem) Init() {
	yfs.files = make(map[string]*YdNoteFile)
}

func (yfs *YdFileSystem) Open(name string) (fs.File, error) {
	if f, ok := yfs.files[name]; ok {
		return f, nil
	}
	return nil, fmt.Errorf("file not found:%s", name)
}

func (yfs *YdFileSystem) ReadDir(name string) ([]fs.DirEntry, error) {
	f, err := yfs.Open(name)
	if err != nil {
		return nil, err
	}
	if stat, _ := f.Stat(); !stat.IsDir() {
		return nil, fmt.Errorf("%s is not dir", name)
	}

	return nil, nil
}
