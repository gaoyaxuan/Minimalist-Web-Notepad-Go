package main

import (
	"github.com/robfig/cron/v3"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func main() {
	port := getEnvOrDefaultString("PORT", "80")
	fileMaxDay := getEnvOrDefaultInt("FILE_MAX_DAY", 30)
	randomStrLen := getEnvOrDefaultInt("STR_LEN", 10)
	spec := getEnvOrDefaultString("SPEC", "0 0 1 * * ?")

	// 设置 Gin 为生产模式
	gin.SetMode(gin.ReleaseMode)
	// 创建一个默认的路由引擎
	router := gin.New()
	// 使用日志中间件和恢复中间件
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	router.Static("/static", "./static")
	router.LoadHTMLGlob("index.html")

	// 当访问根目录时，生成一个随机字符串，并重定向到"/random"路径
	router.GET("/", func(c *gin.Context) {
		randomStr := randomString(randomStrLen)
		c.Redirect(http.StatusFound, "/"+randomStr)
	})

	router.GET("/:path", func(c *gin.Context) {
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

		fileContent, err := os.ReadFile(filePath)
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

	router.POST("/:path", func(c *gin.Context) {
		// 读取POST请求的内容
		body, err := io.ReadAll(c.Request.Body)
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
			err := os.Mkdir(dir, 0755)
			if err != nil {
				return
			}
		}

		// 写入文件
		err = os.WriteFile(filePath, body, 0644)
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
		log.Println("执行删除过期文件")
		err := deleteOldFiles("./_tmp_/", fileMaxDay)
		if err != nil {
			return
		}
	}
	//定时任务
	// 添加定时任务,
	_, err := crontab.AddFunc(spec, task)
	if err != nil {
		log.Fatalf("添加定时任务失败: %s\n", err)
		return
	}

	// 启动定时器
	crontab.Start()

	err = router.Run("0.0.0.0:" + port)
	if err != nil {
		log.Fatalf("服务启动失败: %v\n", err)
		return
	} else {
		log.Println("服务启动成功")
	}
}

// randomString 生成指定长度的随机字符串
func randomString(length int) string {
	seed := rand.NewSource(time.Now().UnixNano())
	r := rand.New(seed)
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[r.Intn(len(charset))]
	}
	return string(b)
}

// 递归删除旧文件,
func deleteOldFiles(dirPath string, days int) error {
	// 当前时间减去30天
	cutOffDate := time.Now().AddDate(0, 0, -days)
	emptyFileCutOffDate := time.Now().AddDate(0, 0, -3)

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			// 空文件 超过三天的直接删除(有可能清理的时候空文件正在被使用)
			if info.Size() == 0 && info.ModTime().Before(emptyFileCutOffDate) {
				err := os.Remove(path)
				if err != nil {
					log.Fatalf("failed to remove empty file: %s\n", err)
				}
				// 不是空的需要等到达到的天数
			} else if info.ModTime().Before(cutOffDate) {
				err := os.Remove(path)
				if err != nil {
					log.Fatalf("failed to remove file: %s\n", err)
				}
			}
		}
		//else {
		//	err := deleteOldFiles(path, days)
		//	if err != nil {
		//		return err
		//	}
		//}
		return nil
	})

	if err != nil {
		log.Printf("error walking the path: %v", err)
	}
	return nil
}

// getEnvOrDefaultString 获取环境变量的值，如果未设置则返回默认值
func getEnvOrDefaultString(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// getEnvOrDefaultInt 获取环境变量的值，如果未设置则返回默认的整数值
func getEnvOrDefaultInt(key string, defaultValue int) int {
	strValue := os.Getenv(key)
	if strValue == "" {
		return defaultValue
	}

	value, err := strconv.Atoi(strValue)
	if err != nil {
		log.Printf("环境变量 %s 的值无法转换为整数，默认使用默认值 %d", key, defaultValue)
		return defaultValue
	}

	return value
}
