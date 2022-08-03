package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"io"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/rs/zerolog/log"
)

func HTTPReq(ctx *YDNoteContext, method, httpURL string, data interface{}, timeoutSeconds int) ([]byte, error) {
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
		Jar: ctx.GetHTTPCookies(httpURL),
	}

	var request *http.Request
	var err error
	if data != nil {
		var body []byte
		if b, ok := data.([]byte); ok {
			body = b
		} else {
			body, err = json.Marshal(data)
			if err != nil {
				return nil, err
			}
		}
		buf := bytes.NewBuffer(body)
		// reqBody = buf.Bytes()
		log.Trace().Str("url", httpURL).RawJSON("req", buf.Bytes()).Msg("post data request begin")
		request, err = http.NewRequestWithContext(ctx.Context, method, httpURL, buf)
		request.Header.Set("Content-Type", "application/json")

	} else {
		request, err = http.NewRequestWithContext(ctx.Context, method, httpURL, nil)
		log.Trace().Str("url", httpURL).Msg("post data request begin")
	}
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	log.Trace().Interface("resp_head", resp.Header).Msg("post data resp")
	defer resp.Body.Close()

	response, err := io.ReadAll(resp.Body)
	return response, err
}

func downloadImg(ctx *YDNoteContext, httpURL string, localPath string) (image.Image, error) {
	client := &http.Client{Jar: ctx.GetHTTPCookies(httpURL)}
	request, err := http.NewRequestWithContext(ctx.Context, "GET", httpURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http request fail status:%d", resp.StatusCode)
	}

	var img image.Image
	img, _, err = image.Decode(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("image formt(%s) err(%w)", resp.Header.Get("Content-Type"), err)
	}

	var f *os.File
	f, err = os.OpenFile(localPath, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0755)
	if err != nil {
		return nil, fmt.Errorf("请手动删除cache中的文件(%w)", err)
	}
	err = png.Encode(f, img)
	if err != nil {
		return nil, err
	}
	_ = f.Close()
	return img, nil
}
