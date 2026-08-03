package handlers

import (
	"bufio"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path"
	"time"
	"webserver-arm/log"
	"webserver-arm/models"
	"webserver-arm/zmq"

	"github.com/gin-gonic/gin"
)

var (
	filePre1   = "eco-update-n"
	filePre2   = "update-data"
	uploadDir  = path.Join(os.Getenv("ECOEMS"))
	runExeName = "update.sh"
)

func getEnvParam(name string) string {
	configEnv := os.Getenv(name)
	if configEnv == "" {
		log.Errorf("env %s not set", name)
	}
	return configEnv
}

// @Summary		执行升级命令
// @Description	执行升级命令
// @ID				runupdate
// @Tags			update
// @Accept			json
// @Produce		text/plain
// @Param			Authorization	header		string				true	"JWT Token"	default(<token>)
// @Success		200				{object}	models.CommonResp		"成功 {\"code\": 0, \"msg\": \"\"}"
// @Failure		400				{object}	models.CommonResp	"上传失败 {\"code\": 400, \"msg\": \"\"}"
// @Failure		401				{object}	models.CommonResp	"认证失败 {\"code\": 401, \"msg\": \"\"}"
// @Failure		500				{object}	models.CommonResp	"内部错误 {\"code\": 500, \"msg\": \"\"}"
// @Router			/api/v1/log/update/run [post]
func RunUpdate(c *gin.Context) {
	log.Info("RunUpdate in ..")

	// 创建目标文件
	filePath := path.Join(getEnvParam("ARM_SF"), c.Param("filename"))
	updateFile, err := os.Create(filePath)
	if err != nil {
		log.Error("RunUpdate Create file %s error: %v", filePath, err)
		c.JSON(http.StatusBadRequest, models.CommonResp{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}
	defer updateFile.Close()
	// 读取文件数据流，写入文件
	if _, err := io.Copy(updateFile, c.Request.Body); err != nil {
		log.Error("RunUpdate Read Request.Body error: %v", filePath, err)
		c.JSON(http.StatusBadRequest, models.CommonResp{Code: http.StatusBadRequest, Message: err.Error()})
	}

	// 设置 SSE 响应头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Writer.Flush()

	clientID := fmt.Sprintf("CallProcess_%d", time.Now().UnixMilli())
	zmqClient, err := zmq.NewZmqClient(zmq.ZMQ_SYSTEM_INFO_REQ_ENDPOINT, clientID)
	if err != nil {
		log.Errorf("create zmq client failed: %v", err)
		c.SSEvent("error", err.Error())
		return
	}

	msgCh, err := zmqClient.SendAndLoopReceiveMsg("ota", filePath, clientID[12:])
	if err != nil {
		log.Errorf("zmq client SendAndLoopReceiveMsg failed: %v", err)
		c.SSEvent("error", err.Error())
		return
	}

	// 获取请求的 context，用于检测客户端断开
	ctx := c.Request.Context()

	for {
		select {
		case msg, ok := <-msgCh:
			if !ok {
				// 通道已关闭，结束推送
				log.Info("msgCh closed, exit")
				return
			}
			// 将消息通过 SSE 发送
			if len(msg) < 3 {
				err := fmt.Errorf("升级程序返回结果数量错误")
				c.SSEvent("error", err.Error())
				return
			}
			if msg[1] == "ota_progress" { // 进度
				b := []byte(msg[2])
				c.SSEvent("progress", binary.LittleEndian.Uint32(b[:4]))
			} else if msg[1] == "ota_result" { // 升级结果, 第4位为失败原因
				if len(msg) == 4 {
					c.SSEvent("error", msg[3])
					return
				}
			} else {
				c.SSEvent("error", fmt.Sprintf("错误的消息类型: %s", msg[1]))
				return
			}

			c.Writer.Flush() // 确保立即发送

		case <-ctx.Done():
			// 客户端断开连接，停止循环并退出
			log.Info("client disconnected, stop streaming")
			return
		}
	}
}

func Backup(c *gin.Context) {
	// 1. 设置 SSE 响应头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*") // 如需跨域
	c.Writer.Flush()

	armSFPath := getEnvParam("ARM_SF")
	scriptPath := path.Join(armSFPath, "backup_data.sh")

	// 2. 创建带超时的命令上下文
	ctx, cancel := context.WithTimeout(c.Request.Context(), 60*time.Minute)
	defer cancel()

	filename := fmt.Sprintf("backup_%s", time.Now().Format("20060102_150405"))
	cmd := exec.CommandContext(ctx, scriptPath, "-o", filename)

	// 3. 获取管道，实时读取输出
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		c.SSEvent("error", fmt.Sprintf("获取stdout管道失败: %v", err))
		return
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		c.SSEvent("error", fmt.Sprintf("获取stderr管道失败: %v", err))
		return
	}

	// 4. 启动备份脚本
	if err := cmd.Start(); err != nil {
		c.SSEvent("error", fmt.Sprintf("启动脚本失败: %v", err))
		return
	}

	// 6. 合并 stdout 和 stderr 的输出（使用单个 scanner）
	outputCh := make(chan string)
	go func() {
		// 合并两个 Reader，先读完 stdout 再读 stderr
		merged := io.MultiReader(stdout, stderr)
		scanner := bufio.NewScanner(merged)
		for scanner.Scan() {
			outputCh <- scanner.Text()
		}
		if err := scanner.Err(); err != nil {
			outputCh <- fmt.Sprintf("[ERROR] 读取输出失败: %v", err)
		}
		close(outputCh) // 输出结束后关闭 channel
	}()

	// 7. 发送开始事件
	c.SSEvent("start", "备份脚本已启动")
	c.Writer.Flush()

	// 8. 主循环：处理输出和命令结束
	for {
		select {
		case line, ok := <-outputCh:
			if !ok {
				// outputCh 已关闭，说明脚本输出结束，但进程可能还未退出, 等待进程退出并获取最终状态
				if err := cmd.Wait(); err != nil {
					c.SSEvent("error", fmt.Sprintf("脚本执行失败: %v", err))
				} else {
					c.SSEvent("done", "备份完成")
					c.SSEvent("filename", filename+".tar.gz")
				}
				c.Writer.Flush()
				return
			}
			// 实时发送输出
			c.SSEvent("log", line)
			c.Writer.Flush()
		case <-ctx.Done():
			// 超时或客户端断开
			reason := "执行超时或连接关闭"
			if ctx.Err() == context.DeadlineExceeded {
				reason = "备份执行超时(30分钟)"
			}
			c.SSEvent("error", reason)
			c.Writer.Flush()
			// 尝试终止进程
			if cmd.Process != nil {
				cmd.Process.Kill()
			}
			return
		}
	}
}

// 下载文件
func DownloadFile(c *gin.Context) {
	log.Info("DownloadHisDataFile in ...")
	filename := c.Param("filename")

	armSFPath := getEnvParam("ARM_SF")
	filaPath := path.Join(armSFPath, filename)

	_, err := os.Stat(filaPath)
	if err != nil {
		log.Errorf("DownloadHisDataFile: get file failed: %w", err)
		c.JSON(http.StatusInternalServerError, models.CommonResp{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}

	defer os.Remove(filaPath)

	c.Header("Content-Type", "application/zip")
	c.FileAttachment(filaPath, filaPath)
}
