package agent

import (
	"encoding/json"
	"fmt"

	"github.com/a13labs/cobot/internal/nlp"
)

type InteractionMessages struct {
	Context string
	Content string
	Type    int
}

type LLMAgent struct {
	LLMClient   *nlp.LLMClient
	SystemInput string
}

type LLMStringResult struct {
	Result string `json:"result"`
}

type LLMIntListResult struct {
	Result []int `json:"result"`
}

func NewLLMAgent(llmClient *nlp.LLMClient, agentName string) *LLMAgent {
	return &LLMAgent{
		LLMClient:   llmClient,
		SystemInput: fmt.Sprintf("You are %s, a machine acting as a layer between a system and a user. You recieve instructions. You generate short messages. The messages are to the point. One line only. The output is in JSON format.", agentName),
	}
}

func (i *LLMAgent) MessageRequest(instruction string) (string, error) {

	llmMessage, err := i.LLMClient.SendChatNoStream([]nlp.LLMChatMessage{
		{
			Role:    "System",
			Content: fmt.Sprintf("%s Format: { \"result\" : \"\" }.", i.SystemInput),
		},
		{
			Role:    "User",
			Content: instruction,
		},
	})
	if err != nil {
		return "", err
	}
	jsonResult := LLMStringResult{}
	err = json.Unmarshal([]byte(llmMessage.Message.Content), &jsonResult)
	if err != nil {
		return "", nil
	}
	return jsonResult.Result, nil
}

func (i *LLMAgent) IntListRequest(instruction string) ([]int, error) {

	llmMessage, err := i.LLMClient.SendChatNoStream([]nlp.LLMChatMessage{
		{
			Role:    "System",
			Content: fmt.Sprintf("%s Format: { \"result\" : [] }.", i.SystemInput),
		},
		{
			Role:    "User",
			Content: instruction,
		},
	})
	if err != nil {
		return nil, err
	}
	jsonResult := LLMIntListResult{}
	err = json.Unmarshal([]byte(llmMessage.Message.Content), &jsonResult)
	if err != nil {
		return nil, nil
	}
	return jsonResult.Result, nil
}

func (i *LLMAgent) Embeddings(text string) ([]float64, error) {
	embeddings, err := i.LLMClient.EmbeddingRequest(&nlp.LLMEmbeddingRequest{Prompt: text})
	if err != nil {
		return nil, err
	}

	return embeddings, nil
}
