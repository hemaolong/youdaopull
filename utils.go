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
	"golang.org/x/net/publicsuffix"
)

func HTTPReq(ctx *YDNoteContext, method, httpURL string, data interface{}, timeoutSeconds int) ([]byte, error) {
	var jar http.CookieJar
	var err error
	if len(ctx.Cookies.Cookies) > 0 {
		var cs []*http.Cookie
		for _, v := range ctx.Cookies.Cookies {
			cs = append(cs, &http.Cookie{Name: v.Name,
				Value:  v.Value,
				Path:   v.Path,
				Domain: v.Domain,
			})

			// fmt.Println(">>>>", v.Name, v.Value)
		}
		cookieURL, _ := url.Parse(httpURL)

		jar, _ = cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
		jar.SetCookies(cookieURL, cs)
	}
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
		request.Header.Set("Content-Type", "application/json")

	} else {
		request, err = http.NewRequestWithContext(ctx.Context, "POST", httpURL, nil)
		log.Trace().Str("url", httpURL).Msg("post data request begin")
	}
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(request)
	log.Trace().Interface("resp_head", resp.Header).Msg("post data resp")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	response, err := io.ReadAll(resp.Body)
	return response, err
}
