package routers

import (
	"embed"
	"fmt"
	"github.com/yumenaka/comi/common"
	"github.com/yumenaka/comi/routers/reverse_proxy"
	"html/template"
	"math/rand"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

//退出时清理
func init() {
	common.SetupCloseHander()
}

func StartServer(args []string) {
	initBaseMode()
	cmdPath := path.Dir(os.Args[0]) //去除路径最后一个元素  /home/dir/comigo.exe -> /home/dir/
	if len(args) == 0 {
		err := common.ScanBookPath(cmdPath)
		if err != nil {
			fmt.Println("扫描出错，执行目录：", cmdPath)
		}
	} else {
		for _, p := range args {
			if p == cmdPath {
				continue //指定参数的话，就不扫描当前目录
			}
			err := common.ScanBookPath(p)
			if err != nil {
				fmt.Println("扫描出错，扫描路径：", p)
			}
		}
	}
	switch len(common.BookList) {
	case 0:
		fmt.Println("没找到可阅读书籍，程序退出。")
		os.Exit(0)
	default:
		setFirstBook(args)
	}
	var wg sync.WaitGroup
	//解压图片，分析分辨率
	if common.Config.CheckImageInServer {
		wg.Add(1)
		go func() {
			common.InitReadingBook()
			defer wg.Done()
		}()
		wg.Wait()
	} else {
		err := common.InitReadingBook()
		if err != nil {
			fmt.Println("无法初始化书籍。", err, common.ReadingBook)
			//os.Exit(0)
		}
	}
	InitWebServer()
}

func initBaseMode() {





	// 当前执行目录
	runPath,_ := os.Getwd()
	fmt.Println("runPath =", runPath)
	// 带后缀的执行文件名
	filenameWithSuffix := path.Base(os.Args[0])
	// 文件后缀
	fileSuffix := path.Ext(filenameWithSuffix)
	// 去掉后缀后的执行文件名
	filenameWithOutSuffix := strings.TrimSuffix(filenameWithSuffix, fileSuffix)
	fmt.Println("filenameWithOutSuffix =", filenameWithOutSuffix)
	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}
	extPath := filepath.Dir(ex)
	fmt.Println(extPath)
	ExtFileName:=  strings.Trim(filenameWithOutSuffix, extPath)
	fmt.Println("ExtFileName =", ExtFileName)
}



func setFirstBook(args []string) {
	if len(common.BookList) == 0 {
		return
	}
	//多本书，读第一本
	if len(args) == 0 {
		if len(common.BookList) > 0 {
			common.ReadingBook = common.BookList[0]
		}
	}
	if len(args) > 0 {
		for _, b := range common.BookList {
			if b.FilePath == args[0] {
				common.ReadingBook = b
				break
			}
		}
	}
}

//启动web服务
func InitWebServer() {
	//设置 gin
	engine := gin.Default()
	if common.Config.LogToFile {
		// 关闭 log 打印的字体颜色。输出到文件不需要颜色
		gin.DisableConsoleColor()
		// 输出 log 到文件(logrus)
		engine.Use(common.LoggerToFile())
	}
	//自定义分隔符，避免与vue.js冲突
	engine.Delims("[[", "]]")

	//静态资源(web.EmbedFiles)
	//go:embed index.html
	var TemplateString string
	//go:embed  favicon.ico js/* css/*
	var EmbedFiles embed.FS
	//网站图标
	engine.GET("favicon.ico", func(c *gin.Context) {
		file, _ := EmbedFiles.ReadFile("favicon.ico")
		c.Data(
			http.StatusOK,
			"image/x-icon",
			file,
		)
	})
	engine.StaticFS("/assets",http.FS(EmbedFiles))
	if common.ReadingBook.IsFolder {
		engine.StaticFS("/raw/"+common.ReadingBook.Name, gin.Dir(common.ReadingBook.FilePath, true))
	} else {
		engine.StaticFile("/raw/"+common.ReadingBook.Name, common.ReadingBook.FilePath)
	}
	//获取模板，命名为"template-data"，同时把左右分隔符改为 [[ ]]
	tmpl := template.Must(template.New("template-data").Delims("[[", "]]").Parse(TemplateString))
	//使用模板
	engine.SetHTMLTemplate(tmpl)
	//解析模板到HTML
	engine.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "template-data", gin.H{
			"title": common.ReadingBook.Name, //页面标题
		})
	})
	//解析json
	engine.GET("/book.json", func(c *gin.Context) {
		c.PureJSON(http.StatusOK, common.ReadingBook)
	})
	//解析书架json
	engine.GET("/bookshelf.json", func(c *gin.Context) {
		c.PureJSON(http.StatusOK, common.BookList)
	})
	//初始化websocket
	engine.GET("/ws", wsHandler)
	//是否同时对外服务
	webHost := ":"
	if common.Config.DisableLAN {
		webHost = "localhost:"
	}
	//检测端口
	if !common.CheckPort(common.Config.Port) {
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		if common.Config.Port+2000 > 65535 {
			common.Config.Port = common.Config.Port + r.Intn(1024)
		} else {
			common.Config.Port = 50000 + r.Intn(10000)
		}
		fmt.Println("端口被占用，尝试随机端口:" + strconv.Itoa(common.Config.Port))
	}
	//webp反向代理
	if common.Config.EnableWebpServer {
		webpError := common.StartWebPServer(common.PictureDir, common.PictureDir, common.TempDir+"/webp", common.Config.Port+1)
		if webpError != nil {
			fmt.Println("无法启动webp转换服务,请检查命令格式，并确认PATH里面有webp-server可执行文件", webpError.Error())
			engine.Static("/cache", common.PictureDir)
		} else {
			fmt.Println("webp转换服务已启动")
			engine.Use(reverse_proxy.ReverseProxyHandle("/cache", reverse_proxy.ReverseProxyOptions{
				TargetHost:  "http://localhost",
				TargetPort:  strconv.Itoa(common.Config.Port + 1),
				RewritePath: "/cache",
			}))
		}
	} else {
		//图片目录
		engine.Static("/cache", common.PictureDir)
	}
	if common.Config.EnableFrpcServer {
		if common.Config.FrpConfig.RandomRemotePort {
			r := rand.New(rand.NewSource(time.Now().UnixNano()))
			common.Config.FrpConfig.RemotePort = 40000 + r.Intn(10000)
		} else {
			if common.Config.FrpConfig.RemotePort <= 0 || common.Config.FrpConfig.RemotePort > 65535 {
				common.Config.FrpConfig.RemotePort = common.Config.Port
			}
		}
		frpcError := common.StartFrpC(common.TempDir)
		if frpcError != nil {
			fmt.Println("无法启动frpc服务,请检查命令格式，并确认PATH里面有frpc可执行文件", frpcError.Error())
		} else {
			fmt.Println("frpc已启动")
		}
	}
	//开始服务
	common.PrintAllReaderURL()
	//打印配置
	fmt.Println("打印配置")
	fmt.Println(common.Config)
	err := engine.Run(webHost + strconv.Itoa(common.Config.Port))
	if err != nil {
		fmt.Fprintf(os.Stderr, "web服务启动失败，端口: %q\n", common.Config.Port)
	}
}
