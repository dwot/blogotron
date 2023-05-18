package openai

import (
	"context"
	"encoding/base64"
	"errors"
	"golang/util"
	"os"
)

import (
	openai "github.com/sashabaranov/go-openai"
)

func GenerateArticle(useGpt4 bool, prompt string, systemPrompt string) (article string, err error) {
	article, err = generate(useGpt4, prompt, systemPrompt)
	return
}

func GenerateTitle(useGpt4 bool, article string, prompt string, systemPrompt string) (title string, err error) {
	title, err = generate(useGpt4, prompt, systemPrompt, article)
	util.Logger.Info().Msg("Generated title: " + title)
	return
}

func GenerateImg(p string) []byte {
	client := openai.NewClient(os.Getenv("OPENAI_API_KEY"))
	ctx := context.Background()
	reqBase64 := openai.ImageRequest{
		Prompt:         p,
		Size:           openai.CreateImageSize256x256,
		ResponseFormat: openai.CreateImageResponseFormatB64JSON,
		N:              1,
	}
	respBase64, err := client.CreateImage(ctx, reqBase64)
	util.HandleError(err, "Error generating image")
	if err != nil {
		return nil
	}

	imgBytes, err := base64.StdEncoding.DecodeString(respBase64.Data[0].B64JSON)
	util.HandleError(err, "Error decoding image")
	if err != nil {
		return nil
	}

	return imgBytes
}

func generate(useGpt4 bool, prompt string, systemPrompt string, article ...string) (string, error) {
	client := openai.NewClient(os.Getenv("OPENAI_API_KEY"))
	model := openai.GPT3Dot5Turbo
	if useGpt4 {
		model = openai.GPT4
	}
	var messages []openai.ChatCompletionMessage
	messages = append(messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleSystem,
		Content: systemPrompt,
	})
	if len(article) > 0 {
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleAssistant,
			Content: article[0],
		})
	}
	messages = append(messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: prompt,
	})
	resp, err := client.CreateChatCompletion(context.Background(), openai.ChatCompletionRequest{
		Model:    model,
		Messages: messages,
	})

	if err != nil {
		util.HandleError(err, "Error generating article")
		return "", err
	}

	if resp.Choices[0].FinishReason != "stop" {
		err = errors.New("ChatCompletion error (FinishReason): " + resp.Choices[0].FinishReason)
		util.HandleError(err, "Error generating article")
		return "", err
	}

	return resp.Choices[0].Message.Content, nil
}
