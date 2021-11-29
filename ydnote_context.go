/*
 * @Author: maolong.he@gmail.com
 * @Date: 2021-11-26 11:47:32
 * @Last Modified by: maolong.he@gmail.com
 * @Last Modified time: 2021-11-26 11:57:21
 */

package main

import (
	"context"
	"net/http"
	"net/http/cookiejar"
	"net/url"

	"github.com/chromedp/cdproto/network"
	"github.com/rs/zerolog/log"
	"golang.org/x/net/publicsuffix"
)

type YDNoteContext struct {
	// localDir  string
	// remoteDir string // 开始拉取的远程目录
	// smmsSecretToken string
	Context       context.Context
	ContextCancel func()

	Cookies network.SetCookiesParams
}

func CreateContext() *YDNoteContext {
	ctx, cancel := context.WithCancel(context.TODO())
	ydContext = ctx
	return &YDNoteContext{Context: ctx, ContextCancel: cancel}
}

func (yc *YDNoteContext) GetHTTPCookies(host string) http.CookieJar {
	if yc.Cookies.Cookies == nil || len(yc.Cookies.Cookies) == 0 {
		return nil
	}

	var cs []*http.Cookie
	for _, v := range yc.Cookies.Cookies {
		cs = append(cs, &http.Cookie{Name: v.Name,
			Value:  v.Value,
			Path:   v.Path,
			Domain: v.Domain,
		})

		// fmt.Println(">>>>", v.Name, v.Value)
	}
	cookieURL, err := url.Parse(host)
	if err != nil {
		log.Error().Err(err).Msg("convert cookie err")
		return nil
	}

	jar, _ := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	jar.SetCookies(cookieURL, cs)
	return jar
}
