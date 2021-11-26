package main

import (
	"bytes"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/rs/zerolog/log"
	"github.com/tidwall/gjson"
	"golang.org/x/net/publicsuffix"
)

// POST with json request/response
func HTTPReqJSON(ctx *YDNoteContext, method, httpURL string, data interface{}, timeoutSeconds int) (*gjson.Result, error) {
	resp, err := HTTPReq(ctx, method, httpURL, data, timeoutSeconds)
	if err != nil {
		return nil, err
	}
	ret := gjson.ParseBytes(resp)
	return &ret, nil
}

func HTTPReq(ctx *YDNoteContext, method, httpURL string, data interface{}, timeoutSeconds int) ([]byte, error) {
	var cs []*http.Cookie
	for _, v := range ctx.Cookies.Cookies {
		cs = append(cs, &http.Cookie{Name: v.Name,
			Value:  v.Value,
			Path:   v.Path,
			Domain: v.Domain,
		})
	}
	cookieURL, _ := url.Parse(httpURL)

	var jar *cookiejar.Jar
	var err error
	jar, err = cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	jar.SetCookies(cookieURL, cs)
	client := &http.Client{Timeout: time.Duration(timeoutSeconds) * time.Second,
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   time.Duration(timeoutSeconds) * time.Second,
				KeepAlive: time.Duration(timeoutSeconds) * time.Second,
				DualStack: true,
			}).DialContext,
			// MaxIdleConns:          tc.threadCnt,
			// MaxIdleConnsPerHost:   tc.threadCnt,
			IdleConnTimeout:       time.Duration(timeoutSeconds) * time.Second,
			TLSHandshakeTimeout:   time.Duration(timeoutSeconds) * time.Second,
			ExpectContinueTimeout: time.Duration(timeoutSeconds) * time.Second,
		},
		Jar: jar,
	}

	var request *http.Request
	// var reqBody []byte
	if data != nil {
		var body []byte
		body, err := jsoniter.Marshal(data)
		if err != nil {
			return nil, err
		}
		buf := bytes.NewBuffer(body)
		// reqBody = buf.Bytes()
		log.Trace().Str("url", httpURL).RawJSON("req", buf.Bytes()).Msg("post data request begin")
		request, err = http.NewRequestWithContext(ctx.Context, "POST", httpURL, buf)
	} else {
		request, err = http.NewRequestWithContext(ctx.Context, "POST", httpURL, nil)
		log.Trace().Str("url", httpURL).Msg("post data request begin")
	}

	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	response, err := io.ReadAll(resp.Body)
	return response, err
}
