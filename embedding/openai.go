package embedding

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// OpenAIClient OpenAI Embedding客户端
type OpenAIClient struct {
	baseURL    string
	apiKey     string
	model      string
	httpClient *http.Client
}

// NewOpenAIClient 创建OpenAI客户端
// baseURL: API基础URL，默认为 https://api.openai.com/v1
// apiKey: OpenAI API密钥
// model: 使用的模型，默认为 text-embedding-3-small
func NewOpenAIClient(baseURL, apiKey, model string) *OpenAIClient {
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	if model == "" {
		model = "text-embedding-3-small"
	}

	return &OpenAIClient{
		baseURL: baseURL,
		apiKey:  apiKey,
		model:   model,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// NewOpenAIClientFromConfig 从配置创建OpenAI客户端
func NewOpenAIClientFromConfig(config Config) *OpenAIClient {
	return NewOpenAIClient(config.BaseURL, config.APIKey, config.Model)
}

// CreateEmbedding 为单个文本创建向量嵌入
func (c *OpenAIClient) CreateEmbedding(text string) ([]float32, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("OpenAI API key is not set")
	}

	// 构建请求体
	requestBody := map[string]interface{}{
		"input": text,
		"model": c.model,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// 构建HTTP请求
	url := fmt.Sprintf("%s/embeddings", c.baseURL)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

	// 发送请求
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// 检查HTTP状态码
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OpenAI API error (status %d): %s", resp.StatusCode, string(body))
	}

	// 解析响应
	var response EmbeddingResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(response.Data) == 0 {
		return nil, fmt.Errorf("no embedding returned from API")
	}

	return response.Data[0].Embedding, nil
}

// CreateEmbeddingFunc 返回一个EmbeddingFunc，使用OpenAI客户端
func (c *OpenAIClient) CreateEmbeddingFunc() EmbeddingFunc {
	return func(text string) ([]float32, error) {
		return c.CreateEmbedding(text)
	}
}

// CreateBatchEmbeddings 为多个文本批量创建向量嵌入
func (c *OpenAIClient) CreateBatchEmbeddings(texts []string) ([][]float32, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("OpenAI API key is not set")
	}

	if len(texts) == 0 {
		return [][]float32{}, nil
	}

	// 构建请求体
	requestBody := map[string]interface{}{
		"input": texts,
		"model": c.model,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// 构建HTTP请求
	url := fmt.Sprintf("%s/embeddings", c.baseURL)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

	// 发送请求
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// 检查HTTP状态码
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OpenAI API error (status %d): %s", resp.StatusCode, string(body))
	}

	// 解析响应
	var response EmbeddingResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(response.Data) == 0 {
		return nil, fmt.Errorf("no embeddings returned from API")
	}

	// 按照原始顺序提取向量
	embeddings := make([][]float32, len(texts))
	for _, item := range response.Data {
		if item.Index >= 0 && item.Index < len(texts) {
			embeddings[item.Index] = item.Embedding
		}
	}

	return embeddings, nil
}

// CreateBatchEmbeddingFunc 返回一个BatchEmbeddingFunc，使用OpenAI客户端
func (c *OpenAIClient) CreateBatchEmbeddingFunc() BatchEmbeddingFunc {
	return func(texts []string) ([][]float32, error) {
		return c.CreateBatchEmbeddings(texts)
	}
}

// EmbeddingResponse OpenAI API响应结构
type EmbeddingResponse struct {
	Object string `json:"object"`
	Data   []struct {
		Object    string    `json:"object"`
		Embedding []float32 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
	Model string `json:"model"`
	Usage struct {
		PromptTokens int `json:"prompt_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage"`
}
