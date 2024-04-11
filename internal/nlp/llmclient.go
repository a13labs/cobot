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

type LLMStringResult struct {
	Result string `json:"result"`
}

type LLMIntListResult struct {
	Result []int `json:"result"`
}

type LLMStringListResult struct {
	Result []string `json:"result"`
}
type LLMBoolResult struct {
	Result bool `json:"result"`
}

var llmJSONSystemConstraints = []string{
	"-Receive instructions.",
	"-Follow provided rules.",
	"-Execute instructions.",
	"-Generate short responses.",
	"-Generate only what is asked.",
	"-Write response in raw JSON only.",
}

var llmJSONUserConstraints = []string{
	"-One line response.",
	"-Don't explain results.",
	"-Don't provide examples.",
	"-Follow the schema.",
	"-Generate what is asked.",
}

func composeJSONSystemInput(schema string) string {
	constrainsList := strings.Join(llmJSONSystemConstraints, "\n")
	return fmt.Sprintf("%s\n-Write the response using the JSON schema:'%s'.", constrainsList, schema)
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

func (llm *LLMClient) RequestChat(messages []LLMChatMessage) (*LLMChatResponseNoStream, error) {

	url := fmt.Sprintf("http://%s:%d/api/chat", llm.Host, llm.Port)

	requestBodyBytes, err := json.Marshal(messages)
	if err != nil {
		return nil, err
	}

	request_body := fmt.Sprintf(`{"model": "%s", "messages": %s, "stream" : false, "format" : "json"}`, llm.Model, string(requestBodyBytes))

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

func (llm *LLMClient) RequestCompletion(request *LLMCompletionRequest) (*LLMCompletionResponseNoStream, error) {

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

func (llm *LLMClient) MessageRequest(instructions string) (string, error) {

	schema := "{\"result\":string}"
	msg, err := llm.JSONRequest(schema, instructions)
	if err != nil {
		return "", err
	}
	jsonResult := LLMStringResult{}
	err = json.Unmarshal([]byte(msg), &jsonResult)
	if err != nil {
		return "", nil
	}
	return jsonResult.Result, nil
}

func (llm *LLMClient) IntListRequest(instructions string) ([]int, error) {

	schema := "{\"result\":[int]}"
	msg, err := llm.JSONRequest(schema, instructions)
	if err != nil {
		return nil, err
	}
	jsonResult := LLMIntListResult{}
	err = json.Unmarshal([]byte(msg), &jsonResult)
	if err != nil {
		return nil, nil
	}
	return jsonResult.Result, nil
}

func (llm *LLMClient) StringListRequest(instructions string) ([]string, error) {

	schema := "{\"result\":[string]}"
	msg, err := llm.JSONRequest(schema, instructions)
	if err != nil {
		return nil, err
	}
	jsonResult := LLMStringListResult{}
	err = json.Unmarshal([]byte(msg), &jsonResult)
	if err != nil {
		return nil, nil
	}
	return jsonResult.Result, nil
}

func (llm *LLMClient) BoolRequest(instructions string) (bool, error) {

	schema := "{\"result\":boolean}"
	msg, err := llm.JSONRequest(schema, instructions)
	if err != nil {
		return false, err
	}
	jsonResult := LLMBoolResult{}
	err = json.Unmarshal([]byte(msg), &jsonResult)
	if err != nil {
		return false, nil
	}
	return jsonResult.Result, nil
}

func (llm *LLMClient) JSONRequest(schema string, instructions string) (string, error) {
	llmMessage, err := llm.RequestChat([]LLMChatMessage{
		{
			Role:    "System",
			Content: composeJSONSystemInput(schema),
		},
		{
			Role:    "User",
			Content: fmt.Sprintf("Instructions:\n%s", instructions),
		},
		{
			Role:    "User",
			Content: fmt.Sprintf("Rules:\n%s", strings.Join(llmJSONUserConstraints, "\n")),
		},
		{
			Role:    "User",
			Content: "Response:",
		},
	})

	if err != nil {
		return "", err
	}

	return llmMessage.Message.Content, nil
}
