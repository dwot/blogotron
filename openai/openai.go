package openai

import (
	"context"
	"encoding/base64"
	"errors"
	"golang/util"
	"strconv"
	"strings"
	"time"
)

import (
	openai "github.com/sashabaranov/go-openai"
)

var TopicTemplate = "Come up with {{.IdeaCount}} new and different topics to write multiple blog posts about. {{.IdeaConcept}}  Each of your new topics should be completely different from the provided existing topics but appeal to the same general audience and be appropriate for the same blog.  The should be broad categories that can be used to drive ideas for many blog articles."
var KeywordTemplate = "You are an SEO expert. Given the article idea \"{{.Prompt}}\", suggest a relevant and strong primary keyword that aligns with this topic. Consider the potential search intent of users interested in this topic, and aim for a keyword that is specific, has a good balance between search volume and competition, and accurately represents the main focus of the article."
var IdeaTemplate = "Come up with {{.IdeaCount}} new ideas for articles. {{.IdeaConcept}}"

func GenerateArticle(apiKey string, useGpt4 bool, prompt string, systemPrompt string) (article string, err error) {
	hardArticleRules := ""
	article, err = generate(apiKey, useGpt4, prompt+hardArticleRules, systemPrompt)
	util.Logger.Info().Msg("Generated article: " + strconv.Itoa(len(article)) + " characters")
	return
}

func GenerateTopics(apiKey string, useGpt4 bool, prompt string, systemPrompt string) (topics string, err error) {
	hardTopicRules := " Return the results as a single line, unnumbered Pipe-Delimitted list. Each topic should not be encapsulated in quotation marks."
	topics, err = generate(apiKey, useGpt4, prompt+hardTopicRules, systemPrompt)
	util.Logger.Info().Msg("Generated topics: " + topics)
	return
}

func GenerateIdeas(apiKey string, useGpt4 bool, prompt string, systemPrompt string) (ideas string, err error) {
	hardIdeaRules := " Return the results as a single line, unnumbered Pipe-Delimitted list. Each idea should not be encapsulated in quotation marks."
	ideas, err = generate(apiKey, useGpt4, prompt+hardIdeaRules, systemPrompt)
	util.Logger.Info().Msg("Generated ideas: " + ideas)
	return
}

func GenerateTestGreeting(apiKey string, useGpt4 bool, prompt string, systemPrompt string) (greeting string, err error) {
	greeting, err = generate(apiKey, useGpt4, prompt, systemPrompt)
	util.Logger.Info().Msg("Generated keyword: " + greeting)
	return
}

func GenerateKeywords(apiKey string, useGpt4 bool, prompt string, systemPrompt string) (keyword string, err error) {
	hardKeywordRules := " Return the keyword alone, no other text or markup."
	keyword, err = generate(apiKey, useGpt4, prompt+hardKeywordRules, systemPrompt)
	util.Logger.Info().Msg("Generated keyword: " + keyword)
	return
}

func GenerateDescription(apiKey string, useGpt4 bool, article string, prompt string, systemPrompt string) (description string, err error) {
	hardDescriptionRules := " Return the description alone, no other text or markup.  Do not include any new keywords just the body of the description itself."
	description, err = generate(apiKey, useGpt4, prompt+hardDescriptionRules, systemPrompt, article)
	util.Logger.Info().Msg("Generated Description: " + description)
	return
}
func GenerateImagePrompt(apiKey string, useGpt4 bool, article string, prompt string, systemPrompt string) (imgPrompt string, err error) {
	hardImagePromptRules := ""
	imgPrompt, err = generate(apiKey, useGpt4, prompt+hardImagePromptRules, systemPrompt, article)
	util.Logger.Info().Msg("Generated image prompt: " + imgPrompt)
	return
}
func GenerateTitle(apiKey string, useGpt4 bool, article string, prompt string, systemPrompt string) (title string, err error) {
	hardTitleRules := ""
	title, err = generate(apiKey, useGpt4, prompt+hardTitleRules, systemPrompt, article)
	util.Logger.Info().Msg("Generated title: " + title)
	return
}
func GenerateImageSearch(apiKey string, useGpt4 bool, article string, prompt string, systemPrompt string) (imgSearch string, err error) {
	hardImageSearchRules := ""
	imgSearch, err = generate(apiKey, useGpt4, prompt+hardImageSearchRules, systemPrompt, article)
	util.Logger.Info().Msg("Generated Image Search: " + imgSearch)
	return
}

func GenerateImg(p string, apiKey string) ([]byte, error) {
	client := openai.NewClient(apiKey)
	ctx := context.Background()
	reqBase64 := openai.ImageRequest{
		Prompt:         p,
		Size:           openai.CreateImageSize256x256,
		ResponseFormat: openai.CreateImageResponseFormatB64JSON,
		N:              1,
	}
	respBase64, err := client.CreateImage(ctx, reqBase64)
	if err != nil {
		return nil, err
	}

	imgBytes, err := base64.StdEncoding.DecodeString(respBase64.Data[0].B64JSON)
	if err != nil {
		return nil, err
	}

	return imgBytes, nil
}

func generate(apiKey string, useGpt4 bool, prompt string, systemPrompt string, article ...string) (string, error) {
	client := openai.NewClient(apiKey)
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

	maxRetries := 3
	retries := 0
	success := false

	for !success && retries < maxRetries {
		resp, err := client.CreateChatCompletion(context.Background(), openai.ChatCompletionRequest{
			Model:    model,
			Messages: messages,
		})
		if err != nil {
			if strings.Contains(err.Error(), "429") {
				util.Logger.Info().Msg("API returned busy, waiting 5 seconds")
				time.Sleep(5 * time.Second)
				retries++
				continue
			} else {
				return "", err
			}
		} else {
			success = true
			return resp.Choices[0].Message.Content, nil
		}
		if resp.Choices[0].FinishReason != "stop" {
			err = errors.New("ChatCompletion error (FinishReason): " + resp.Choices[0].FinishReason)
			return "", err
		}
	}

	if !success {
		err := errors.New("API busy and max retries met, please try again later")
		return "", err
	}
	return "", errors.New("Unknown error")
}
