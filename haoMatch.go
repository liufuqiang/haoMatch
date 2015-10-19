package main

import (
	"./convtrad"
	"./darts"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const checkAppendSec = 3
const MAX_CATE_NUM = 1024

var checkAppendTime = checkAppendSec * time.Second //seconds

type dartMap map[string]darts.Darts

func dartsInit(cate string) (dmap dartMap, err error) { // {{{
	dmap = make(dartMap, MAX_CATE_NUM)
	if cate == "" {
		fileArr := findFiles()
		for _, cate_file := range fileArr {
			if cate_file == "" {
				continue
			}
			d, err := darts.Import("./"+cate_file, "./"+strings.Replace(cate_file, "_", "-", -1)+".lib")
			if err != nil {
				log.Println("ERROR: ", cate_file, " darts initial failed!")
			} else {
				log.Println("INFO: ", cate_file, " darts initial success!")
			}
			dmap[cate_file] = d
		}
	} else {
		d, err := darts.Import("./"+cate, "./"+strings.Replace(cate, "_", "-", -1)+".lib")
		if err != nil {
			log.Println("ERROR: ", cate, " darts initial failed!")
		} else {
			log.Println("INFO: ", cate, " darts initial success!")
		}
		dmap[cate] = d
	}
	return dmap, err
} // }}}

var dartmap, err = dartsInit("")
var convTrad = convtrad.New()

func haoMatch(w http.ResponseWriter, r *http.Request) { // {{{

	arg_text := strings.TrimSpace(r.FormValue("text"))
	arg_cate := strings.TrimSpace(r.FormValue("cate"))
	if arg_text == "" || arg_cate == "" {
		fmt.Fprintf(w, "")
		return
	}

	//繁体转简体
	arg_text = convTrad.ToSimp(arg_text)
	//全角转半角
	arg_text = darts.SBC2DBC(arg_text)
	text := []rune(arg_text)

	cate := arg_cate
	cateName := "cate_" + cate + ".txt"
	result := ""

	if dart, ok := dartmap[cateName]; ok && len(text) > 0 {
		dart = dartmap[cateName]
		if len(text) > 0 {
			textLen := len(text)
			for j := 0; j < textLen; j++ {
				for i := j; i < textLen; i++ {
					newKey := text[j : i+1]
					results := dart.Search(newKey, 0)
					for _, item := range results {
						if string(newKey) == string(item.Key) {
							result += string(item.Key) + "\t" + strconv.Itoa(item.Value) + "\n"
							j = i + 1
						}
					}
				}
			}
		}
	}
	fmt.Fprintf(w, result) //这个写入到w的是输出到客户端的
	return
} // }}}

func appendData() {
	for {
		fileArr := findFiles()
		for _, file := range fileArr {
			fileinfo, e := os.Stat(file)
			if e != nil {
				continue
			}
			if time.Now().Unix()-fileinfo.ModTime().Unix() >= checkAppendSec {
				continue
			}

			var dartmapAppend, err = dartsInit(file)
			if err == nil {
				dartmap = dartmapAppend
			}
		}
		time.Sleep(checkAppendTime)
	}
}

func main() { // {{{

	go appendData()                      //增量流程
	http.HandleFunc("/match/", haoMatch) //设置访问的路由

	err := http.ListenAndServe(":9093", nil) //设置监听的端口
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
} // }}}

func findFiles() []string {
	fileArr := make([]string, MAX_CATE_NUM)
	files, _ := ioutil.ReadDir("./")
	for i, file := range files {
		if file.IsDir() {
			continue
		} else {
			reg := regexp.MustCompile(`(?U)^cate_.*\.txt`)
			fname := reg.FindString(file.Name())
			if fname != "" {
				fileArr[i] = fname
			}
		}
	}
	return fileArr
}
