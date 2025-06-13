package login

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"image/color"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/skip2/go-qrcode"
	"github.com/tidwall/gjson"
)

func getTvQrcodeUrlAndAuthCode() (string, string) {
	api := "https://passport.bilibili.com/x/passport-tv-login/qrcode/auth_code"
	data := make(map[string]string)
	data["local_id"] = "0"
	data["ts"] = fmt.Sprintf("%d", time.Now().Unix())
	signature(&data)
	dataString := strings.NewReader(mapToString(data))
	client := http.Client{}
	req, _ := http.NewRequest("POST", api, dataString)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	code := gjson.Parse(string(body)).Get("code").Int()
	if code == 0 {
		qrcodeUrl := gjson.Parse(string(body)).Get("data.url").String()
		authCode := gjson.Parse(string(body)).Get("data.auth_code").String()
		return qrcodeUrl, authCode
	} else {
		panic("get_tv_qrcode_url_and_auth_code error")
	}
}

func verifyLogin(authCode string, cookiePath string) {
	api := "http://passport.bilibili.com/x/passport-tv-login/qrcode/poll"
	data := make(map[string]string)
	data["auth_code"] = authCode
	data["local_id"] = "0"
	data["ts"] = fmt.Sprintf("%d", time.Now().Unix())
	signature(&data)
	dataString := strings.NewReader(mapToString(data))
	client := http.Client{}
	req, _ := http.NewRequest("POST", api, dataString)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	for {
		resp, err := client.Do(req)
		if err != nil {
			panic(err)
		}
		body, _ := io.ReadAll(resp.Body)
		code := gjson.Parse(string(body)).Get("code").Int()
		if code == 0 {
			fmt.Println("登录成功")
			if cookiePath == "" {
				cookiePath = "cookie.json"
			} else {
				// 检查文件夹是否存在
				if _, err := os.Stat(cookiePath); os.IsNotExist(err) {
					// 创建文件夹
					err := os.MkdirAll(cookiePath, 0755)
					if err != nil {
						panic(err)
					}
				}
			}
			err := os.WriteFile(cookiePath, []byte(string(body)), 0644)
			if err != nil {
				panic(err)
			}
			fmt.Println("cookie 已保存在", cookiePath)
			break
		} else {
			time.Sleep(time.Second * 3)
		}
		resp.Body.Close()
	}
}

var appkey = "4409e2ce8ffd12b8"
var appsec = "59b43e04ad6965f34319062b478f83dd"

func signature(params *map[string]string) {
	var keys []string
	(*params)["appkey"] = appkey
	for k := range *params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var query string
	for _, k := range keys {
		query += k + "=" + url.QueryEscape((*params)[k]) + "&"
	}
	query = query[:len(query)-1] + appsec
	hash := md5.New()
	hash.Write([]byte(query))
	(*params)["sign"] = hex.EncodeToString(hash.Sum(nil))
}

func mapToString(params map[string]string) string {
	var query string
	for k, v := range params {
		query += k + "=" + v + "&"
	}
	query = query[:len(query)-1]
	return query
}

func checkOrMakeCookieFile(cookiePath string) error {
	// 检查文件夹是否存在
	if _, err := os.Stat(cookiePath); os.IsNotExist(err) {
		// 创建文件夹
		err := os.MkdirAll(path.Dir(cookiePath), 0755)
		if err != nil {
			return err
		}
		// 创建 cookie.json 文件
		_, err = os.Create(cookiePath)
		if err != nil {
			return err
		}
	}
	return nil
}

func LoginBili(cookiePath string) (loginUrl string) {

	checkOrMakeCookieFile(cookiePath)

	loginUrl, authCode := getTvQrcodeUrlAndAuthCode()

	fmt.Println("请在手机B站扫描二维码登录")
	qr, err := qrcode.New(loginUrl, qrcode.Medium)
	if err != nil {
		panic(err)
	}
	fmt.Println(qr.ToSmallString(false))
	fmt.Println("或将此链接复制到手机B站打开:", loginUrl)
	// 同步输出到文件
	imagePath := path.Join(path.Dir(cookiePath), "qrcode.png")
	if _, err := os.Stat(imagePath); os.IsNotExist(err) {
		// 创建文件
		_, err := os.Create(imagePath)
		if err != nil {
			panic(err)
		}
	}
	qr.ForegroundColor = color.RGBA{0, 0, 0, 255}       // 黑色前景
	qr.BackgroundColor = color.RGBA{255, 255, 255, 255} // 白色背景

	// 生成PNG
	pngData, err := qr.PNG(256)
	if err != nil {
		panic(err)
	}
	err = os.WriteFile(imagePath, pngData, 0644)
	if err != nil {
		panic(err)
	}
	fmt.Println("二维码已同步保存到:", imagePath)
	defer verifyLogin(authCode, cookiePath)
	return loginUrl
}
