package nlp

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
)

type LLMClient struct {
	Host  string
	Port  int
	Model string
}

type LLMChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type LLMChatResponseNoStream struct {
	Model              string         `json:"model"`
	CreatedAt          string         `json:"created_at"`
	Message            LLMChatMessage `json:"message"`
	Done               bool           `json:"done"`
	TotalDuration      int            `json:"total_duration"`
	LoadDuration       int            `json:"load_duration"`
	PromptEvalCount    int            `json:"prompt_eval_count"`
	PromptEvalDuration int            `json:"prompt_eval_duration"`
	EvalCount          int            `json:"eval_count"`
	EvalDuration       int            `json:"eval_duration"`
}

type LLMCompletionRequest struct {
	Prompt string `json:"prompt"`
}

type LLMCompletionResponseNoStream struct {
	Model              string `json:"model"`
	CreatedAt          string `json:"created_at"`
	Response           string `json:"response"`
	Done               bool   `json:"done"`
	Context            []int  `json:"context"`
	TotalDuration      int    `json:"total_duration"`
	LoadDuration       int    `json:"load_duration"`
	PromptEvalCount    int    `json:"prompt_eval_count"`
	PromptEvalDuration int    `json:"prompt_eval_duration"`
	EvalCount          int    `json:"eval_count"`
	EvalDuration       int    `json:"eval_duration"`
}

type LLMEmbeddingRequest struct {
	Prompt string `json:"prompt"`
}

type LLMEmbeddingResponse struct {
	Embeddings []float64 `json:"embedding"`
}

func NewLLMClient(host string, port int, model string) *LLMClient {

	llm := &LLMClient{
		Host:  host,
		Port:  port,
		Model: model,
	}

	if !llm.HealthCheck() {
		return nil
	}

	// Get the model info from the server endpoint /api/show

	url := fmt.Sprintf("http://%s:%d/api/show", llm.Host, llm.Port)
	request_body := fmt.Sprintf(`{"name": "%s"}`, llm.Model)
	resp, err := http.Post(url, "application/json", strings.NewReader(request_body))

	if err != nil {
		return nil
	}

	if resp == nil || resp.StatusCode != http.StatusOK {
		return nil
	}

	defer resp.Body.Close()

	return llm
}

func (llm *LLMClient) HealthCheck() bool {
	// Check if the LLMClient server is running but opening a connection to it
	conn := fmt.Sprintf("%s:%d", llm.Host, llm.Port)
	_, err := net.Dial("tcp", conn)
	return err == nil
}

func (llm *LLMClient) SendChatNoStream(messages []LLMChatMessage) (*LLMChatResponseNoStream, error) {

	url := fmt.Sprintf("http://%s:%d/api/chat", llm.Host, llm.Port)

	requestBodyBytes, err := json.Marshal(messages)
	if err != nil {
		return nil, err
	}

	request_body := fmt.Sprintf(`{"model": "%s", "messages": %s, "stream" : false }`, llm.Model, string(requestBodyBytes))

	resp, err := http.Post(url, "application/json", strings.NewReader(request_body))

	if err != nil {
		return nil, err
	}

	if resp == nil || resp.StatusCode != http.StatusOK {
		return nil, errors.New("error getting response from LLMClient server")
	}

	defer resp.Body.Close()

	// Read the response from the server
	msg := &LLMChatResponseNoStream{}
	err = json.NewDecoder(resp.Body).Decode(&msg)
	if err != nil {
		return nil, err
	}

	return msg, nil
}

func (llm *LLMClient) SendCompletion(request *LLMCompletionRequest) (*LLMCompletionResponseNoStream, error) {

	url := fmt.Sprintf("http://%s:%d/api/generate", llm.Host, llm.Port)

	request_body := fmt.Sprintf(`{"model": "%s", "prompt": "%s"}`, llm.Model, request.Prompt)
	resp, err := http.Post(url, "application/json", strings.NewReader(request_body))

	if err != nil {
		return nil, err
	}

	if resp == nil || resp.StatusCode != http.StatusOK {
		return nil, errors.New("error getting response from LLMClient server")
	}

	defer resp.Body.Close()

	// Read the response from the server
	msg := &LLMCompletionResponseNoStream{}
	err = json.NewDecoder(resp.Body).Decode(&msg)
	if err != nil {
		return nil, err
	}

	return msg, nil
}

func (llm *LLMClient) EmbeddingRequest(request *LLMEmbeddingRequest) ([]float64, error) {

	url := fmt.Sprintf("http://%s:%d/api/embeddings", llm.Host, llm.Port)

	request_body := fmt.Sprintf(`{"model": "%s", "prompt": "%s"}`, llm.Model, request.Prompt)
	resp, err := http.Post(url, "application/json", strings.NewReader(request_body))

	if err != nil {
		return nil, err
	}

	if resp == nil || resp.StatusCode != http.StatusOK {
		return nil, errors.New("error getting response from LLMClient server")
	}

	defer resp.Body.Close()

	// Read the response from the server
	msg := &LLMEmbeddingResponse{}
	err = json.NewDecoder(resp.Body).Decode(&msg)
	if err != nil {
		return nil, err
	}

	return msg.Embeddings, nil
}
