package embedding

import (
	"testing"
)

func TestMockEmbeddingFunc(t *testing.T) {
	dim := 384
	fn := MockEmbeddingFunc(dim)

	// 测试相同文本生成相同向量
	text1 := "hello world"
	vec1, err := fn(text1)
	if err != nil {
		t.Fatalf("MockEmbeddingFunc failed: %v", err)
	}

	if len(vec1) != dim {
		t.Errorf("expected dimension %d, got %d", dim, len(vec1))
	}

	vec2, err := fn(text1)
	if err != nil {
		t.Fatalf("MockEmbeddingFunc failed: %v", err)
	}

	// 验证相同文本生成相同向量
	for i := 0; i < dim; i++ {
		if vec1[i] != vec2[i] {
			t.Errorf("vectors differ at index %d: %f != %f", i, vec1[i], vec2[i])
		}
	}

	// 测试不同文本生成不同向量
	text2 := "goodbye world"
	vec3, err := fn(text2)
	if err != nil {
		t.Fatalf("MockEmbeddingFunc failed: %v", err)
	}

	different := false
	for i := 0; i < dim; i++ {
		if vec1[i] != vec3[i] {
			different = true
			break
		}
	}

	if !different {
		t.Error("different texts should generate different vectors")
	}
}

func TestRandomEmbeddingFunc(t *testing.T) {
	dim := 384
	fn := RandomEmbeddingFunc(dim)

	text := "hello world"
	vec1, err := fn(text)
	if err != nil {
		t.Fatalf("RandomEmbeddingFunc failed: %v", err)
	}

	if len(vec1) != dim {
		t.Errorf("expected dimension %d, got %d", dim, len(vec1))
	}

	// 验证向量已归一化
	norm := float32(0)
	for _, v := range vec1 {
		norm += v * v
	}

	if norm < 0.99 || norm > 1.01 {
		t.Errorf("vector not normalized: norm = %f", norm)
	}

	// 测试每次调用生成不同向量
	vec2, err := fn(text)
	if err != nil {
		t.Fatalf("RandomEmbeddingFunc failed: %v", err)
	}

	different := false
	for i := 0; i < dim; i++ {
		if vec1[i] != vec2[i] {
			different = true
			break
		}
	}

	if !different {
		t.Error("RandomEmbeddingFunc should generate different vectors on each call")
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.BaseURL == "" {
		t.Error("BaseURL should not be empty")
	}

	if config.Model == "" {
		t.Error("Model should not be empty")
	}

	// APIKey可以为空（需要用户设置）
}

func TestNewOpenAIClient(t *testing.T) {
	// 测试默认参数
	client := NewOpenAIClient("", "", "")
	if client == nil {
		t.Fatal("NewOpenAIClient returned nil")
	}

	if client.baseURL != "https://api.openai.com/v1" {
		t.Errorf("expected default baseURL, got %s", client.baseURL)
	}

	if client.model != "text-embedding-3-small" {
		t.Errorf("expected default model, got %s", client.model)
	}

	// 测试自定义参数
	client2 := NewOpenAIClient("https://custom.api.com/v1", "test-key", "custom-model")
	if client2.baseURL != "https://custom.api.com/v1" {
		t.Errorf("expected custom baseURL, got %s", client2.baseURL)
	}

	if client2.apiKey != "test-key" {
		t.Errorf("expected custom apiKey, got %s", client2.apiKey)
	}

	if client2.model != "custom-model" {
		t.Errorf("expected custom model, got %s", client2.model)
	}
}

func TestNewOpenAIClientFromConfig(t *testing.T) {
	config := Config{
		BaseURL: "https://test.api.com/v1",
		APIKey:  "test-key",
		Model:   "test-model",
	}

	client := NewOpenAIClientFromConfig(config)
	if client == nil {
		t.Fatal("NewOpenAIClientFromConfig returned nil")
	}

	if client.baseURL != config.BaseURL {
		t.Errorf("expected baseURL %s, got %s", config.BaseURL, client.baseURL)
	}

	if client.apiKey != config.APIKey {
		t.Errorf("expected apiKey %s, got %s", config.APIKey, client.apiKey)
	}

	if client.model != config.Model {
		t.Errorf("expected model %s, got %s", config.Model, client.model)
	}
}

func TestOpenAIClientCreateEmbeddingFunc(t *testing.T) {
	client := NewOpenAIClient("", "", "")
	fn := client.CreateEmbeddingFunc()

	if fn == nil {
		t.Fatal("CreateEmbeddingFunc returned nil")
	}

	// 测试函数签名
	_, err := fn("test")
	// 应该失败，因为没有API key
	if err == nil {
		t.Error("expected error when API key is not set")
	}
}
