package controller

import (
	"encoding/json"
	"fmt"
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
	c.Writer.Header().Set("Transfer-Encoding", "chunked")
	c.Writer.Header().Set("Content-Type", "application/json")

	// 模拟多轮对话
	for i := 0; i < 5; i++ {
		// 创建符合要求的JSON结构
		response := map[string]interface{}{
			"id":      "chatcmpl-GfxWG6bNyLU8dKZRVaERvxe5DgJXZ",
			"object":  "chat.completion.chunk",
			"created": time.Now().Unix(),
			"model":   "text_model",
			"choices": []map[string]interface{}{
				{
					"index": i,
					"delta": map[string]string{
						"role":    "assistant",
						"content": "这是第" + fmt.Sprint(i+1) + "条消息",
					},
					"logprobs":      nil,
					"finish_reason": nil,
				},
			},
			"system_fingerprint": nil,
		}

		// 将map转换为JSON字符串
		jsonResponse, err := json.Marshal(response)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to marshal JSON"})
			return
		}

		// 发送JSON响应
		c.Writer.Write(jsonResponse)
		c.Writer.Write([]byte("\n")) // 添加换行符以分隔每个chunk

		// 刷新缓冲区，确保消息即时发送
		c.Writer.Flush()

		// 模拟延迟
		time.Sleep(1 * time.Second)
	}

	// 完成流式输出
	c.AbortWithStatus(http.StatusNoContent)
}

// MdHandle 返回简单流式md文档
func MdHandle(c *gin.Context) {
	// 启用流式输出
	c.Writer.Header().Set("Transfer-Encoding", "chunked")
	c.Writer.Header().Set("Content-Type", "application/json")

	// 从网络获取Markdown文档
	markdownDoc, err := fetchMarkdownDocFromNetwork()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch markdown document"})
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
		response := map[string]interface{}{
			"id":      "chatcmpl-GfxWG6bNyLU8dKZRVaERvxe5DgJXZ",
			"object":  "chat.completion.chunk",
			"created": time.Now().Unix(),
			"model":   "md_model",
			"choices": []map[string]interface{}{
				{
					"index": 0,
					"delta": map[string]string{
						"role":    "assistant",
						"content": markdownDoc[start:end],
					},
					"logprobs":      nil,
					"finish_reason": nil,
				},
			},
			"system_fingerprint": nil,
		}

		// 将map转换为JSON字符串
		jsonResponse, err := json.Marshal(response)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to marshal JSON"})
			return
		}

		// 发送JSON响应
		c.Writer.Write(jsonResponse)
		c.Writer.Write([]byte("\n")) // 添加换行符以分隔每个chunk

		// 刷新缓冲区，确保消息即时发送
		c.Writer.Flush()
	}

	// 完成流式输出
	c.AbortWithStatus(http.StatusNoContent)
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
