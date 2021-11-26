/*
 * @Author: maolong.he@gmail.com
 * @Date: 2021-11-26 11:47:32
 * @Last Modified by: maolong.he@gmail.com
 * @Last Modified time: 2021-11-26 11:57:21
 */

package main

import (
	"context"

	"github.com/chromedp/cdproto/network"
)

type YDNoteContext struct {
	Cstk string

	// localDir  string
	// remoteDir string // 开始拉取的远程目录
	// smmsSecretToken string
	Context       context.Context
	ContextCancel func()

	Cookies network.SetCookiesParams
}

func CreateContext() *YDNoteContext {
	ctx, cancel := context.WithCancel(context.TODO())

	return &YDNoteContext{Context: ctx, ContextCancel: cancel}
}
