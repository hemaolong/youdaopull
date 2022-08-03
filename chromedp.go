package main

import (
	"context"
	"fmt"
	"image"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/chromedp/cdproto"
	"github.com/chromedp/cdproto/browser"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/target"
	"github.com/chromedp/chromedp"
	"github.com/makiuchi-d/gozxing"
	"github.com/makiuchi-d/gozxing/qrcode"
	"github.com/mdp/qrterminal/v3"
	"github.com/rs/zerolog/log"
)

func getQRCode(ydContext *YDNoteContext, sel string) chromedp.ActionFunc {
	return func(ctx context.Context) (err error) {
		var imgSrc string
		// 等待获取url
		for i := 0; i < 100; i++ {
			var nodes []*cdp.Node
			if err = chromedp.Nodes(sel, &nodes, chromedp.NodeReady, chromedp.NodeEnabled).Do(ctx); err != nil {
				return
			}

			imgSrc = nodes[0].AttributeValue("src")
			if len(imgSrc) != 0 {
				log.Info().Str("src", imgSrc).Int("loop", i).Msg("获取到二维码地址")
				break
			}
			time.Sleep(time.Millisecond * 10)
		}

		// 下载图片到本地
		img, err := downloadImg(ydContext, imgSrc, localCacheDir(pngFile))
		if err != nil {
			return err
		}
		log.Info().Str("file", localCacheDir(pngFile)).Msg("下载图片完成")
		if err = printQRCode(img); err != nil {
			return err
		}

		log.Info().Msg("等待扫码...")

		return
	}
}

// 加载Cookies
func loadCookies(ydContext *YDNoteContext, cf string) chromedp.ActionFunc {
	return func(ctx context.Context) (err error) {
		// 如果cookies临时文件不存在则直接跳过
		if _, inErr := os.Stat(cf); os.IsNotExist(inErr) {
			return
		}

		// 如果存在则读取cookies的数据
		cookiesData, err := ioutil.ReadFile(cf)
		if err != nil {
			return
		}

		// 反序列化
		if err = ydContext.Cookies.UnmarshalJSON(cookiesData); err != nil {
			return
		}
		log.Info().Msg("加载到cookie")

		// 设置cookies
		return network.SetCookies(ydContext.Cookies.Cookies).Do(ctx)
	}
}

func saveAndUseCookies(ydContext *YDNoteContext, sel string) chromedp.ActionFunc {
	return func(ctx context.Context) (err error) {
		cookies, err := network.GetAllCookies().Do(ctx)
		log.Info().Err(err).Msg("开始获取cookie")
		if err != nil {
			return
		}

		// 2. 序列化
		cookiesData, err := network.GetAllCookiesReturns{Cookies: cookies}.MarshalJSON()
		if err != nil {
			return
		}

		// 3. 存储到临时文件
		log.Info().Err(err).Str("file", cookieFile).RawJSON("cookie", cookiesData).Msg("存储cookie")
		if err = os.WriteFile(localCacheDir(cookieFile), cookiesData, 0755); err != nil {
			return
		}

		err = loadCookies(ydContext, localCacheDir(cookieFile)).Do(ctx)
		if err != nil {
			return
		}

		if len(sel) > 0 {
			err = chromedp.Click(sel).Do(ctx)
			log.Info().Err(err).Msg("关闭扫码界面")
			if err != nil {
				return
			}
		}
		return
	}
}

func printQRCode(img image.Image) (err error) {
	// 使用gozxing库解码图片获取二进制位图
	bmp, err := gozxing.NewBinaryBitmapFromImage(img)
	if err != nil {
		err = fmt.Errorf("二维码-gozxing解码错误(%w)", err)
		return
	}

	// 用二进制位图解码获取gozxing的二维码对象
	res, err := qrcode.NewQRCodeReader().Decode(bmp, nil)
	if err != nil {
		err = fmt.Errorf("二维码-qrcode解码错误(%w)", err)
		return
	}

	log.Info().Str("qrcode", res.String()).Msg("qrcode str")
	// config := qrterminal.Config{
	// 	Level:     qrterminal.M,
	// 	Writer:    os.Stdout,
	// 	BlackChar: qrterminal.BLACK,
	// 	WhiteChar: qrterminal.WHITE,
	// 	QuietZone: 1,
	// }
	// if runtime.GOOS == "windows" {
	// 	config.Writer = colorable.NewColorableStdout()
	// 	// config.BlackChar = qrterminal.BLACK
	// 	// config.WhiteChar = qrterminal.WHITE
	// }
	qrterminal.Generate(res.String(), qrterminal.M, terminalWriter)
	return
}

func isWXLogin() bool {
	return ydLoginMode == "wx"
}

func isMobileVerificationCodeLogin() bool {
	return ydLoginMode == "mvc"
}

// 登陆
func doLogin(ydContext *YDNoteContext, url string) chromedp.ActionFunc {
	return func(ctx context.Context) (err error) {
		err = loadCookies(ydContext, localCacheDir(cookieFile)).Do(ctx)
		if err != nil {
			return
		}

		err = chromedp.Navigate(url).Do(ctx)
		if err != nil {
			return
		}

		err = chromedp.Sleep(time.Second).Do(ctx)
		if err != nil {
			return
		}

		var url string
		if err = chromedp.Evaluate(`window.location.href`, &url).Do(ctx); err != nil {
			return
		}
		log.Info().Str("url", url).Msg("window.location.href")

		// 不是登陆界面，说明已经登陆成功
		if !strings.Contains(url, `https://note.youdao.com/signIn`) {
			log.Info().Msg("使用cookies登陆成功")
			chromedp.Stop()
			return
		}

		log.Info().Msg("没有cookie，开始登陆...")
		// 不在登陆状态，使用二维码登陆
		if isWXLogin() {
			if err = doQRCodeLogin(ydContext).Do(ctx); err != nil {
				return
			}
		} else if isMobileVerificationCodeLogin() {
			if err = doMobileVerificationLogin(ydContext).Do(ctx); err != nil {
				return
			}
		} else {
			if err = doThirdPartyLogin(ydContext).Do(ctx); err != nil {
				return
			}
		}
		return
	}
}

// 二维码登陆
func doQRCodeLogin(ydContext *YDNoteContext) chromedp.ActionFunc {
	return func(ctx context.Context) (err error) {
		err = chromedp.Click(`body > div.bd > div.login-main > div.login-right > div:nth-child(2) > div.weixin.btn.track-btn`).Do(ctx)
		if err != nil {
			return
		}

		// 获得新打开的第一个非空页面-微信登陆（微信登陆另起一个新页面）
		ch := chromedp.WaitNewTarget(ctx, func(info *target.Info) bool {
			if info.URL != "" {
				log.Info().Str("url", info.URL).Msg("微信登陆界面已经打开")
				return true
			}
			return false
		})

		// 等待微信登陆页签打开
		newCtx, cancel := chromedp.NewContext(ctx, chromedp.WithTargetID(<-ch))
		defer cancel()
		if err := chromedp.Run(newCtx,
			hdlQRCode(ydContext, `img.wx-qrcode-img`)); err != nil {
			log.Fatal().Err(err).Msg("")
		}
		return
	}
}

//  手机验证码登陆
func doMobileVerificationLogin(ydContext *YDNoteContext) chromedp.ActionFunc {
	return func(ctx context.Context) (err error) {
		if err := chromedp.Run(ctx,
			chromedp.WaitVisible(loginSel),
			saveAndUseCookies(ydContext, "")); err != nil {
			log.Fatal().Err(err).Msg("")
		}
		return
	}
}

// 二维码登陆
func doThirdPartyLogin(ydContext *YDNoteContext) chromedp.ActionFunc {
	return func(ctx context.Context) (err error) {
		// 获得新打开的第一个非空页面-微信登陆（微信登陆另起一个新页面）
		ch := chromedp.WaitNewTarget(ctx, func(info *target.Info) bool {
			if info.URL != "" {
				log.Info().Str("url", info.URL).Msg("登陆界面已经打开")
				return true
			}
			return false
		})

		// 等待登陆页签打开
		newCtx, cancel := chromedp.NewContext(ctx, chromedp.WithTargetID(<-ch))
		defer cancel()
		if err := chromedp.Run(newCtx,
			hdlThirdPartyLogin(ydContext)); err != nil {
			log.Fatal().Err(err).Msg("")
		}
		return
	}
}

// 处理二维码
func hdlQRCode(ydContext *YDNoteContext, sel string) chromedp.Tasks {
	return chromedp.Tasks{
		chromedp.WaitReady(sel, chromedp.ByQuery),
		// 获取并且打印二维码
		getQRCode(ydContext, sel),
		chromedp.WaitReady("#close", chromedp.ByQuery),
		saveAndUseCookies(ydContext, "#close"),
	}
}

// 手机验证码登陆
func hdlMobileVerificationCodeLogin(ydContext *YDNoteContext) chromedp.Tasks {
	return chromedp.Tasks{
		// 等待关闭按钮
		chromedp.WaitReady("#close", chromedp.ByQuery),
		saveAndUseCookies(ydContext, "#close"),
	}
}

// 处理普通登陆结果
func hdlThirdPartyLogin(ydContext *YDNoteContext) chromedp.Tasks {
	return chromedp.Tasks{
		// 等待关闭按钮
		chromedp.WaitReady("#close", chromedp.ByQuery),
		saveAndUseCookies(ydContext, "#close"),
	}
}

func doDownload(ctx context.Context, ydContext *YDNoteContext, onLoginOk func(context.Context, *YDNoteContext)) chromedp.ActionFunc {
	return func(ctx context.Context) (err error) {
		log.Info().Msg("开始下载...")
		onLoginOk(ctx, ydContext)
		return
	}
}

func doYoudaoNoteLogin(ydContext *YDNoteContext, host string, onLoginOk func(context.Context, *YDNoteContext)) {
	ctx, _ := chromedp.NewExecAllocator(
		ydContext.Context,

		// 以默认配置的数组为基础，覆写headless参数
		// 当然也可以根据自己的需要进行修改，这个flag是浏览器的设置
		append(chromedp.DefaultExecAllocatorOptions[:], chromedp.Flag("headless", ydHeadless))...,
	)
	ctx, _ = chromedp.NewContext(
		ctx,
		// 设置日志方法
		chromedp.WithLogf(log.Printf),
	)

	// 打开有道官网
	_ = localExpordWordDir("word")
	fp, _ := filepath.Abs(localExpordWordDir())

	log.Info().Str("host", host).Msg("begin navigate")
	if err := chromedp.Run(ctx,
		network.Enable(),
		browser.SetDownloadBehavior(browser.SetDownloadBehaviorBehaviorAllow).
			WithDownloadPath(fp).
			WithEventsEnabled(true),
		doLogin(ydContext, host)); err != nil {
		log.Fatal().Err(err).Send()
		return
	}

	chromedp.ListenTarget(ctx, func(v interface{}) {
		{
			// log.Info().Interface("event", v).Msg("listen target")
		}
		switch ev := v.(type) {
		case *network.EventSignedExchangeReceived:
			// log.Info().Str("RequestID", string(ev.RequestID)).Interface("Info", ev.Info).Msg("listen target event signed exhanged received")

		case *network.EventRequestWillBeSent:
			// log.Info().Interface("raw_query", ev.Request).Msg("listen target request be sent")
		case *network.EventResponseReceived:
			// u, err := url.Parse(ev.Response.URL)
			// if err != nil {
			// 	panic(err)
			// }
			// _, err = url.ParseQuery(u.RawQuery)
			// if err != nil {
			// 	panic(err)
			// }

			// log.Info().Str("raw_query", u.RawQuery).Msg("listen target permision")
			// // if signInfo == nil || len(signToken) == 0 {
			// // 	if sign, ok := vals["sign"]; ok {
			// // 		signInfo = &signStruct{}
			// // 		err := json.Unmarshal([]byte(sign[0]), &signInfo)
			// // 		if err != nil {
			// // 			panic(err)
			// // 		}

			// // 		signInfo.UserID = vals["userId"][0]

			// // 		// if s, err := base64.URLEncoding.DecodeString(signToken.Secret); err == nil {
			// // 		// 	signToken.Secret = string(s)
			// // 		// }
			// // 		// if s, err := base64.URLEncoding.DecodeString(signToken.Signature); err == nil {
			// // 		// 	signToken.Signature = string(s)
			// // 		// }
			// // 	}
			// // 	if token, ok := vals["token"]; ok {
			// // 		signToken = token[0]
			// // 	}
			// // }

		case *browser.EventDownloadWillBegin:
			log.Info().Interface("GUID", ev.GUID).Str("SuggestedFilename", ev.SuggestedFilename).Msg("listen target download begin")
			downloadingFiles.Store(ev.GUID, ev.SuggestedFilename)

		case *browser.EventDownloadProgress:
			if ev.State == browser.DownloadProgressStateCanceled {
				log.Info().Str("state", ev.State.String()).Str("GUID", ev.GUID).
					Msg("listen target download canceled")
				if loadVal, ok := downloadingFiles.LoadAndDelete(ev.GUID); ok {
					filename, _ := loadVal.(string)
					if _, relativeOK := downloadFileAbsPath.LoadAndDelete(filename); relativeOK {
						atomic.AddInt32(&downloadingCount, -1)
					}
				}
			} else if ev.State == browser.DownloadProgressStateCompleted {
				log.Info().Str("state", ev.State.String()).Str("GUID", ev.GUID).
					Msg("listen target download finish")

				if loadVal, ok := downloadingFiles.LoadAndDelete(ev.GUID); ok {
					filename, _ := loadVal.(string)
					if relativePath, relativeOK := downloadFileAbsPath.LoadAndDelete(trimFileName(filename)); relativeOK {
						atomic.AddInt32(&downloadingCount, -1)
						cachePath, _ := filepath.Abs(localExpordWordDir(filename))
						targetPath, _ := relativePath.(string)
						log.Info().Str("state", ev.State.String()).Str("GUID", ev.GUID).
							Str("cache_path", cachePath).
							Str("target_path", targetPath).
							Msg("download finish")

						err := os.Rename(cachePath, targetPath)
						if err != nil {
							cachePath = filepath.Clean(cachePath)
							log.Error().Str("cachePath", cachePath).Err(err).Send()
						}
					} else {
						log.Error().Interface("file", filename).Msg("fail to find file relative path")
					}
				}
			}

		case *network.EventRequestWillBeSentExtraInfo:
		// 	log.Info().Interface("event", ev).Msg("listen target request extra info")
		// // if len(authorization) == 0 {
		// // 	if a, ok := ev.Headers["Authorization"].(string); ok {
		// // 		authorization = a
		// // 		log.Info().Str("authorization", authorization).Msg("get authorization")
		// // 	}
		// // }
		case *cdproto.Message:
			if len(sessionID) == 0 {
				sessionID = ev.SessionID.String()
			}
		default:
			// log.Info().Interface("event", v).Msg("listen targetdefault")
		}
	})

	// _ = chromedp.Sleep(time.Hour).Do(ctx)

	// 等待登陆完成
	if err := chromedp.Run(ctx,
		chromedp.WaitVisible(loginSel),
		doDownload(ctx, ydContext, onLoginOk)); err != nil {
		log.Fatal().Err(err).Send()
		return
	}
	time.Sleep(time.Second * 5)

	log.Info().Str("host", host).Msg("end navigate")
}
