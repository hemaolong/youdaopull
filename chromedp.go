package main

import (
	"context"
	"fmt"
	"image"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	_ "image/jpeg"
	"image/png"

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
		resp, err := http.Get(imgSrc) // (ydContext, "GET", imgSrc, nil, 30)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		var img image.Image
		img, _, err = image.Decode(resp.Body)
		if err != nil {
			return err
		}

		var f *os.File
		f, err = os.OpenFile(localCacheDir(pngFile), os.O_TRUNC|os.O_CREATE, 0o0755)
		if err != nil {
			return fmt.Errorf("请手动删除cache中的文件(%w)", err)
		}
		err = png.Encode(f, img)
		if err != nil {
			return err
		}
		_ = f.Close()

		log.Info().Str("file", localCacheDir(pngFile)).Msg("下载图片完成")
		if err = printQRCode(img); err != nil {
			return err
		}

		log.Info().Msg("等待扫码...")

		return
	}
}

// func findCSTK(params []*network.CookieParam) string {
// 	for _, v := range params {
// 		if v.Name == "YNOTE_CSTK" {
// 			return v.Value
// 		}
// 	}
// 	return ""
// }

// func findCSTKEx(params []*network.Cookie) string {
// 	for _, v := range params {
// 		if v.Name == "YNOTE_CSTK" {
// 			return v.Value
// 		}
// 	}
// 	return ""
// }

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
		// ydContext.Cstk = findCSTK(ydContext.Cookies.Cookies)
		log.Info().Str("cstk", ydContext.Cstk).Msg("加载到cookie")

		// 设置cookies
		return network.SetCookies(ydContext.Cookies.Cookies).Do(ctx)
	}
}

func waitScanQRCode(ydContext *YDNoteContext, sel string) chromedp.ActionFunc {
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

		err = chromedp.Click(sel).Do(ctx)
		log.Info().Err(err).Msg("关闭扫码界面")
		if err != nil {
			return
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
	qrterminal.Generate(res.String(), qrterminal.L, os.Stdout)
	return
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

		// 不在登陆状态，使用二维码登陆
		log.Info().Msg("没有cookie，开始微信登陆...")
		if err = doQRCodeLogin(ydContext).Do(ctx); err != nil {
			return
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

// 处理二维码
func hdlQRCode(ydContext *YDNoteContext, sel string) chromedp.Tasks {
	return chromedp.Tasks{
		chromedp.WaitReady(sel, chromedp.ByQuery),
		// 获取并且打印二维码
		getQRCode(ydContext, sel),
		chromedp.WaitReady("#close", chromedp.ByQuery),
		waitScanQRCode(ydContext, "#close"),
	}
}

func doDownload(ydContext *YDNoteContext, onLoginOk func(*YDNoteContext)) chromedp.ActionFunc {
	return func(ctx context.Context) (err error) {
		log.Info().Msg("开始下载...")
		onLoginOk(ydContext)
		return
	}
}

func doYoudaoNoteLogin(ydContext *YDNoteContext, url string, onLoginOk func(*YDNoteContext)) {
	ctx, _ := chromedp.NewExecAllocator(
		ydContext.Context,

		// 以默认配置的数组为基础，覆写headless参数
		// 当然也可以根据自己的需要进行修改，这个flag是浏览器的设置
		append(chromedp.DefaultExecAllocatorOptions[:], chromedp.Flag("headless", false))...,
	)
	ctx, _ = chromedp.NewContext(
		ctx,
		// 设置日志方法
		chromedp.WithLogf(log.Printf),
	)

	// 打开有道官网
	log.Info().Str("url", url).Msg("begin navigate")
	if err := chromedp.Run(ctx,
		doLogin(ydContext, url)); err != nil {
		log.Fatal().Err(err).Send()
		return
	}

	// 等待登陆完成
	if err := chromedp.Run(ctx,
		chromedp.WaitVisible(loginSel),
		doDownload(ydContext, onLoginOk)); err != nil {
		log.Fatal().Err(err).Send()
		return
	}
	time.Sleep(time.Second * 5)

	log.Info().Str("url", url).Msg("end navigate")
}
