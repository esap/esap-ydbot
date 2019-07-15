package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	ydapp "github.com/esap/EntAppSdkGo"
)

var (
	_Buin      int32
	_AppId     string
	_EncAesKey string

	app    *ydapp.MsgApp
	err    error
	remote string
	local  string
	port   string
)

func init() {
	cfg, err := getConfig("esap")
	if err != nil {
		fmt.Println("未找到配置文件")
	} else {
		port = cfg["port"]
		local = cfg["local"]
		remote = cfg["remote"]
		bint, _ := strconv.ParseInt(cfg["buin"], 10, 32)
		_Buin = int32(bint)
		_AppId = cfg["appid"]
		_EncAesKey = cfg["enckey"]
		ydapp.Server_Addr = cfg["yd"] //设置服务器地址
		ydapp.Callback_Url = "/"      //设置回调地址
	}
}
func main() {
	fmt.Println("ydbot is running at:", port)
	app, err = ydapp.NewMsgApp(_Buin, _AppId, _EncAesKey)

	if err != nil {
		log.Println("New app error:", err)
		return
	}

	app.SetReceiver(&esap{})

	http.HandleFunc("/", app.ServeHTTP)
	http.HandleFunc("/p", func(w http.ResponseWriter, req *http.Request) {
		pi := req.FormValue("id")
		picfile := pi + ".jpg"
		http.ServeFile(w, req, picfile)
		return
	})
	log.Fatal(http.ListenAndServe(":"+port, nil))

}

type esap struct{}

func (e *esap) Receive(msg *ydapp.ReceiveMsg) {
	token, expire, err := app.GetToken()
	if err != nil {
		log.Println("Get token error:", err)
		return
	}
	log.Printf("Token: %s, Expire: %d", token, expire) //expire为过期的时间戳，单位秒
	switch msg.MsgType {
	case "text":
		ret, _ := getAnswer(fmt.Sprint(msg.Text["content"]), msg.FromUser, fmt.Sprint(msg.Buin), "")
		if ret != "" {
			e := app.SendTxtMsg(msg.FromUser, "", ret)
			if e != nil {
				fmt.Println("SendTxtMsg err:", e)
			}
		}
	case "image":
		picurl := strconv.FormatInt(time.Now().UnixNano(), 10)
		filename := picurl + ".jpg"
		e := app.DownloadImageSave(fmt.Sprint(msg.Image["media_id"]), "./"+filename)
		if e != nil {
			fmt.Println("download img err:", e)
		}
		defer time.AfterFunc(30*time.Second, func() { os.Remove(filename) })

		ret, _ := getAnswer("图片", msg.FromUser, fmt.Sprint(msg.Buin), url.QueryEscape("http://"+local+":"+port+"/p?id="+picurl))
		if ret != "" {
			e := app.SendTxtMsg(msg.FromUser, "", ret)
			if e != nil {
				fmt.Println("SendTxtMsg err:", e)
			}
		}
	default:
	}
}

func getAnswer(msg string, uid string, robotName string, pic ...string) (string, error) {
	fmt.Println("[api] 尝试应答=>", msg)
	if len(pic) == 0 {
		pic = append(pic, "")
	}

	httpClient := http.Client{Timeout: 20 * time.Second}
	u := remote + robotName + "?userid=" + uid + "&msg=" + url.QueryEscape(msg) + "&pic=" + pic[0]
	resp, err := httpClient.Post(u, "", nil)
	if err != nil {
		fmt.Println("post-err:", err)
		return "", err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}
