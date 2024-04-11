package agent

import (
	"fmt"

	"github.com/a13labs/cobot/internal/nlp"
)

func getEmbeddings(ctx *AgentCtx, text string) ([]float64, error) {
	embeddings, err := ctx.LLMClient.EmbeddingRequest(&nlp.LLMEmbeddingRequest{Prompt: text})
	if err != nil {
		return nil, err
	}

	return embeddings, nil
}

func isItemInList(ctx *AgentCtx, prompt string, items []string) (bool, error) {
	list := ""
	for _, item := range items {
		list += fmt.Sprintf("-'%s'\n", item)
	}
	instr := fmt.Sprintf("Given list:\n%s\nGiven input:'%s'\n.Any item in the given list similar or related to the given input? true or false?", list, prompt)
	msg, err := ctx.LLMClient.BoolRequest(instr)
	if err != nil {
		return false, err
	}
	return msg, nil
}

func filterListItems(ctx *AgentCtx, prompt string, items []string) ([]int, error) {
	list := ""
	for i, item := range items {
		list += fmt.Sprintf("-ID:%d,Text:'%s'\n", i, item)
	}
	instr := fmt.Sprintf("Given list:\n%s\nGiven input:'%s'\n.List all items of the given list which the text is similar or related to what is requested in the given input.Write the IDs of all matched items.", list, prompt)
	msg, err := ctx.LLMClient.IntListRequest(instr)
	if err != nil {
		return nil, err
	}
	return msg, nil
}

func isItAQuestion(ctx *AgentCtx, prompt string) (bool, error) {

	instr := fmt.Sprintf("Given input:'%s'\n.'true' if it is a question, 'false' if not.", prompt)

	msg, err := ctx.LLMClient.BoolRequest(instr)
	if err != nil {
		return false, err
	}
	return msg, nil
}

func generateAMessage(ctx *AgentCtx, prompt string) (string, error) {
	return ctx.LLMClient.MessageRequest(prompt)
}
