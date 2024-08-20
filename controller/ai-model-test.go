package controller

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"io/ioutil"
	"net/http"
	"time"
)

type MyRequestBody struct {
	Model string `json:"model"`
}

// MyTestHandleRequest 简单的根据model来返回不同的内容
func MyTestHandleRequest(c *gin.Context) {

	var requestBody MyRequestBody

	// 读取POST请求的JSON体
	if err := c.BindJSON(&requestBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}
	switch requestBody.Model {
	case "text_model":
		TextHandle(c)
		break
	case "md_model":
		MdHandle(c)
		break
	}
}

// TextHandle 返回简单流式文本
func TextHandle(c *gin.Context) {
	// 启用流式输出
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")

	// 模拟多轮对话
	message := "这是一条测试消息"
	for i, char := range message {
		// 创建符合要求的JSON结构
		choice := map[string]interface{}{
			"index": 0,
			"delta": map[string]string{
				"content": string(char),
			},
			"finish_reason": nil,
		}

		if i == len(message)-1 {
			choice["finish_reason"] = "stop"
		}

		response := map[string]interface{}{
			"id":      "chatcmpl-GfxWG6bNyLU8dKZRVaERvxe5DgJXZ",
			"object":  "chat.completion.chunk",
			"created": time.Now().Unix(),
			"model":   "text_model",
			"choices": []map[string]interface{}{choice},
		}

		// 将map转换为JSON字符串
		jsonResponse, err := json.Marshal(response)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to marshal JSON"})
			return
		}

		// 发送JSON响应
		c.Writer.Write([]byte("data: "))
		c.Writer.Write(jsonResponse)
		c.Writer.Write([]byte("\n\n"))

		// 刷新缓冲区，确保消息即时发送
		c.Writer.Flush()

		// 模拟延迟
		time.Sleep(100 * time.Millisecond)
	}

	// 发送结束标记
	c.Writer.Write([]byte("data: [DONE]\n\n"))
	c.Writer.Flush()
}

// MdHandle 返回简单流式md文档
func MdHandle(c *gin.Context) {
	// 启用流式输出
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")

	// 从网络获取Markdown文档
	markdownDoc, err := fetchMarkdownDocFromNetwork()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取Markdown文档失败"})
		return
	}

	// 定义每段的字符数
	chunkSize := 100

	// 计算总的段数
	totalChunks := (len(markdownDoc) + chunkSize - 1) / chunkSize

	// 模拟多轮对话
	for i := 0; i < totalChunks; i++ {
		// 计算每段的起始和结束位置
		start := i * chunkSize
		end := start + chunkSize
		if end > len(markdownDoc) {
			end = len(markdownDoc)
		}

		// 创建符合要求的JSON结构
		choice := map[string]interface{}{
			"index": 0,
			"delta": map[string]string{
				"content": markdownDoc[start:end],
			},
			"finish_reason": nil,
		}

		if i == totalChunks-1 {
			choice["finish_reason"] = "stop"
		}

		response := map[string]interface{}{
			"id":      "chatcmpl-GfxWG6bNyLU8dKZRVaERvxe5DgJXZ",
			"object":  "chat.completion.chunk",
			"created": time.Now().Unix(),
			"model":   "md_model",
			"choices": []map[string]interface{}{choice},
		}

		// 将map转换为JSON字符串
		jsonResponse, err := json.Marshal(response)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "JSON序列化失败"})
			return
		}

		// 发送JSON响应
		c.Writer.Write([]byte("data: "))
		c.Writer.Write(jsonResponse)
		c.Writer.Write([]byte("\n\n"))

		// 刷新缓冲区，确保消息即时发送
		c.Writer.Flush()

		// 可以添加一个小延迟来模拟网络延迟
		// time.Sleep(50 * time.Millisecond)
	}

	// 发送结束标记
	c.Writer.Write([]byte("data: [DONE]\n\n"))
	c.Writer.Flush()
}

// fetchMarkdownDocFromNetwork 模拟从网络获取Markdown文档
func fetchMarkdownDocFromNetwork() (string, error) {
	// 这里替换为实际的URL
	url := "https://data.clearmind.fun/d/file/md/test1.md"

	// 发送HTTP GET请求
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// 读取响应体
	contents, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// 将字节切片转换为字符串并返回
	return string(contents), nil
}
