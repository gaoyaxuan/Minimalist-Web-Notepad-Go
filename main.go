package main

import (
	"flag"
	"fmt"
	"github.com/robfig/cron/v3"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
)

func main() {
	// 创建一个默认的路由引擎
	r := gin.Default()
	r.Static("/static", "./static")
	r.LoadHTMLGlob("index.html")

	var random_int int

	flag.IntVar(&random_int, "l", 10, "random int long")

	// 当访问根目录时，生成一个随机字符串，并重定向到"/random"路径
	r.GET("/", func(c *gin.Context) {
		randomString := randomString(random_int)
		c.Redirect(http.StatusFound, "/"+randomString)
	})

	r.GET("/:path", func(c *gin.Context) {
		rand.Seed(time.Now().UnixNano())
		path := c.Param("path")
		filePath := "./_tmp_/" + path

		// 检查文件是否存在，如果不存在则创建
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			// 创建目录
			if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": err.Error(),
				})
				return
			}

			// 创建文件
			if _, err := os.Create(filePath); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": err.Error(),
				})
				return
			}
		}

		fileContent, err := ioutil.ReadFile(filePath)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}

		body := string(fileContent)
		c.HTML(http.StatusOK, "index.html", gin.H{
			"title": path,
			"body":  body,
		})
	})

	r.POST("/:path", func(c *gin.Context) {
		// 读取POST请求的内容
		body, err := ioutil.ReadAll(c.Request.Body)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error reading request body"})
			return
		}

		// 获取文件路径
		path := c.Param("path")

		// 定义文件路径
		filePath := "./_tmp_/" + path

		// 创建文件夹，如果不存在的话
		dir := "./_tmp_/"
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			os.Mkdir(dir, 0755)
		}

		// 写入文件
		err = ioutil.WriteFile(filePath, body, 0644)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error writing to file"})
			return
		}

		// 返回成功的响应
		c.JSON(http.StatusOK, gin.H{"status": "Success"})

	})

	// 新建一个定时任务对象,定时清理旧的文件
	// 根据cron表达式进行时间调度，cron可以精确到秒，大部分表达式格式也是从秒开始。
	//精确到秒
	crontab := cron.New(cron.WithSeconds())
	//定义定时器调用的任务函数
	task := func() {
		fmt.Println("执行删除过期文件", time.Now())
		err := deleteOldFiles()
		if err != nil {
			return
		}
	}
	//定时任务
	spec := "0 0 1 * * ?" //cron表达式，每五秒一次
	// 添加定时任务,
	crontab.AddFunc(spec, task)
	// 启动定时器
	crontab.Start()

	var port string

	flag.StringVar(&port, "p", ":80", "port to listen on")

	flag.Parse()

	r.Run(port)
}

// randomString 生成指定长度的随机字符串
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

// 递归删除旧文件
func deleteOldFiles() error {
	// 替换为你的目录路径
	dirPath := "./_tmp_/"
	// 当前时间减去30天
	cutOffDate := time.Now().AddDate(0, 0, -1)

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && (info.Size() == 0 || info.ModTime().Before(cutOffDate)) {
			err := os.Remove(path)
			if err != nil {
				fmt.Printf("failed to remove file: %s\n", err)
			}
		}
		return nil
	})

	if err != nil {
		fmt.Printf("error walking the path: %v\n", err)
	}
	return nil
}
