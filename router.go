package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"github.com/spf13/viper"
	"golang/models"
	"golang/openai"
	"html/template"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
)

type Post struct {
	Title       string `json:"title"`
	Content     string `json:"content"`
	Description string `json:"description"`
	Image       []byte `json:"image"`
	Prompt      string `json:"prompt"`
	ImagePrompt string `json:"image-prompt"`
	Error       string `json:"error"`
	ImageB64    string `json:"image64"`
	Length      int    `json:"article-length"`
}

type PageData struct {
	ErrorCode   string `json:"error-code"`
	GPT4Enabled bool   `json:"gpt4-enabled"`
	IdeaText    string `json:"idea-text"`
	IdeaId      string `json:"idea-id"`
}

type PlanData struct {
	ErrorCode string          `json:"error-code"`
	Ideas     []models.Idea   `json:"ideas"`
	Series    []models.Series `json:"series"`
}

type Prompt struct {
	IdeaCount   string
	IdeaConcept string
}

type SeriesData struct {
	ErrorCode string
	Series    interface{}
	Ideas     []models.Idea
}
type IdeaData struct {
	ErrorCode string
	Idea      interface{}
}

var resultsTpl = template.Must(template.ParseFiles("templates\\results.html", "templates\\base.html"))
var writeTpl = template.Must(template.ParseFiles("templates\\write.html", "templates\\base.html"))
var ideaListTpl = template.Must(template.ParseFiles("templates\\ideaList.html", "templates\\base.html"))
var seriesListTpl = template.Must(template.ParseFiles("templates\\seriesList.html", "templates\\base.html"))
var indexTpl = template.Must(template.ParseFiles("templates\\index.html", "templates\\base.html"))
var ideaTpl = template.Must(template.ParseFiles("templates\\idea.html", "templates\\base.html"))
var seriesTpl = template.Must(template.ParseFiles("templates\\series.html", "templates\\base.html"))

func indexHandler(w http.ResponseWriter, _ *http.Request) {
	indexData := PageData{
		ErrorCode: "",
	}
	buf := &bytes.Buffer{}
	err := indexTpl.Execute(buf, indexData)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_, err = buf.WriteTo(w)
	if err != nil {
		fmt.Println("Err rendering menu:" + err.Error())
	}
}

func writeHandler(w http.ResponseWriter, r *http.Request) {
	ideaId := r.FormValue("ideaId")
	id, err := strconv.Atoi(ideaId)
	if err != nil {
		id = 0
	}
	ideaText := ""
	if id > 0 {
		idea, dbErr := models.GetIdeaById(ideaId)
		if dbErr != nil {
			fmt.Println("Err looking up idea by id:" + dbErr.Error())
		}
		ideaText = idea.IdeaText
	}

	gpt4 := os.Getenv("ENABLE_GPT4")
	gpt4enabled := false
	if gpt4 == "true" {
		gpt4enabled = true
	}
	writeData := PageData{
		ErrorCode:   "",
		GPT4Enabled: gpt4enabled,
		IdeaText:    ideaText,
		IdeaId:      ideaId,
	}
	buf := &bytes.Buffer{}
	err = writeTpl.Execute(buf, writeData)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_, err = buf.WriteTo(w)
	if err != nil {
		fmt.Println("Err rendering menu:" + err.Error())
	}
}

func ideaListHandler(w http.ResponseWriter, _ *http.Request) {
	ideas, err := models.GetOpenIdeas()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	planData := PlanData{
		ErrorCode: "",
		Ideas:     ideas,
		Series:    nil,
	}
	buf := &bytes.Buffer{}
	err = ideaListTpl.Execute(buf, planData)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_, err = buf.WriteTo(w)
	if err != nil {
		fmt.Println("Err rendering menu:" + err.Error())
	}
}

func seriesListHandler(w http.ResponseWriter, _ *http.Request) {
	series, err := models.GetSeries(50)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	planData := PlanData{
		ErrorCode: "",
		Ideas:     nil,
		Series:    series,
	}
	buf := &bytes.Buffer{}
	err = seriesListTpl.Execute(buf, planData)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_, err = buf.WriteTo(w)
	if err != nil {
		fmt.Println("Err rendering menu:" + err.Error())
	}
}

func ideaHandler(w http.ResponseWriter, r *http.Request) {
	var ideaData IdeaData
	ideaId := r.FormValue("ideaId")
	seriesId := r.FormValue("seriesId")
	sid, convErr := strconv.Atoi(seriesId)
	if convErr != nil {
		sid = 0
	}
	id, convErr := strconv.Atoi(ideaId)
	if convErr != nil {
		id = 0
	}
	if id > 0 {
		idea, err := models.GetIdeaById(ideaId)
		if err != nil {
			fmt.Println("Err getting idea:" + err.Error())
		}
		ideaData = IdeaData{
			ErrorCode: "",
			Idea:      idea,
		}
	} else {
		ideaData = IdeaData{
			ErrorCode: "",
			Idea: models.Idea{
				SeriesId: sid,
			},
		}
	}
	buf := &bytes.Buffer{}
	err := ideaTpl.Execute(buf, ideaData)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_, err = buf.WriteTo(w)
	if err != nil {
		fmt.Println("Err rendering menu:" + err.Error())
	}
}

func seriesHandler(w http.ResponseWriter, r *http.Request) {
	var seriesData SeriesData
	seriesId := r.FormValue("seriesId")
	id, convErr := strconv.Atoi(seriesId)
	if convErr != nil {
		id = 0
	}
	if id > 0 {
		series, err := models.GetSeriesById(seriesId)
		if err != nil {
			fmt.Println("Err getting series:" + err.Error())
		}
		ideas, err := models.GetOpenSeriesIdeas(seriesId)
		if err != nil {
			fmt.Println("Err getting series ideas:" + err.Error())
		}
		seriesData = SeriesData{
			ErrorCode: "",
			Series:    series,
			Ideas:     ideas,
		}
	} else {
		seriesData = SeriesData{
			ErrorCode: "",
			Series:    models.Series{},
			Ideas:     nil,
		}
	}

	buf := &bytes.Buffer{}
	err := seriesTpl.Execute(buf, seriesData)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_, err = buf.WriteTo(w)
	if err != nil {
		fmt.Println("Err rendering series:" + err.Error())
	}
}

func aiIdeaHandler(w http.ResponseWriter, r *http.Request) {
	gpt4 := r.FormValue("useGpt4")
	useGpt4 := false
	if gpt4 == "true" {
		useGpt4 = true
	}
	seriesId := r.FormValue("seriesId")
	sid, convErr := strconv.Atoi(seriesId)
	if convErr != nil {
		sid = 0
	}
	ideaConcept := r.FormValue("ideaConcept")
	builtConcept := ideaConcept
	if sid > 0 {
		series, _ := models.GetSeriesById(seriesId)
		if strings.TrimSpace(series.SeriesPrompt) != "" {
			builtConcept = "The topic for the ideas is: \"" + series.SeriesPrompt + "\"."
		}
	} else {
		if strings.TrimSpace(ideaConcept) != "" {
			builtConcept = "The topic for the ideas is: \"" + ideaConcept + "\"."
		}
	}

	ideaCount := r.FormValue("ideaCount")
	if strings.TrimSpace(ideaCount) == "" {
		ideaCount = "10"
	}
	prompt := Prompt{
		IdeaCount:   ideaCount,
		IdeaConcept: builtConcept,
	}
	ideaTmpl := template.Must(template.New("idea-prompt").Parse(viper.GetString("config.prompt.idea-prompt")))
	ideaPrompt := new(bytes.Buffer)
	wpErr := ideaTmpl.Execute(ideaPrompt, prompt)
	if wpErr != nil {
		fmt.Println("Err rendering idea prompt:" + wpErr.Error())
	}
	fmt.Println("Prompt is: ", ideaPrompt.String())
	ideaResp, err := openai.GenerateArticle(useGpt4, ideaPrompt.String(), viper.GetString("config.prompt.system-prompt"))
	if err != nil {
		fmt.Println("Err rendering idea response:" + err.Error())
	}
	ideaResp = strings.ReplaceAll(ideaResp, "\n", "")
	fmt.Println("Idea Brainstorm Results: " + ideaResp)
	ideaList := strings.Split(ideaResp, "|")
	for index, value := range ideaList {
		fmt.Printf("Index: %d, Value: %s\n", index, value)
		if strings.TrimSpace(value) != "" {
			idea := models.Idea{
				IdeaText:    strings.TrimSpace(value),
				Status:      "NEW",
				IdeaConcept: ideaConcept,
				SeriesId:    sid,
			}
			_, err = models.AddIdea(idea)
		}
	}
	if sid > 0 {
		seriesHandler(w, r)
	} else {
		ideaListHandler(w, r)
	}
}

func ideaSaveHandler(w http.ResponseWriter, r *http.Request) {
	ideaText := r.FormValue("ideaText")
	ideaId := r.FormValue("ideaId")
	seriesId := r.FormValue("seriesId")
	sid, convErr := strconv.Atoi(seriesId)
	if convErr != nil {
		sid = 0
	}
	id, convErr := strconv.Atoi(ideaId)
	if convErr != nil {
		id = 0
	}
	if id > 0 {
		//Update by Id
		idea := models.Idea{
			Id:       id,
			IdeaText: ideaText,
			Status:   "NEW",
			SeriesId: sid,
		}
		_, err := models.UpdateIdea(idea, id)
		if err != nil {
			fmt.Println("Err saving idea:" + err.Error())
		}
	} else {
		//Insert New
		idea := models.Idea{
			IdeaText: ideaText,
			Status:   "NEW",
			SeriesId: sid,
		}
		_, err := models.AddIdea(idea)
		if err != nil {
			fmt.Println("Err saving idea:" + err.Error())
		}
	}
	if sid > 0 {
		seriesHandler(w, r)
	} else {
		ideaListHandler(w, r)
	}
}

func seriesSaveHandler(w http.ResponseWriter, r *http.Request) {
	seriesName := r.FormValue("seriesName")
	seriesPrompt := r.FormValue("seriesPrompt")
	seriesId := r.FormValue("seriesId")
	id, convErr := strconv.Atoi(seriesId)
	if convErr != nil {
		id = 0
	}
	if id > 0 {
		//Update by Id
		series := models.Series{
			Id:           id,
			SeriesName:   seriesName,
			SeriesPrompt: seriesPrompt,
		}
		_, err := models.UpdateSeries(series, id)
		if err != nil {
			fmt.Println("Err saving series:" + err.Error())
		}
	} else {
		//Insert New
		series := models.Series{
			SeriesName:   seriesName,
			SeriesPrompt: seriesPrompt,
		}
		_, err := models.AddSeries(series)
		if err != nil {
			fmt.Println("Err saving series:" + err.Error())
		}
	}
	seriesHandler(w, r)
}

func ideaRemoveHandler(w http.ResponseWriter, r *http.Request) {
	ideaId := r.FormValue("ideaId")
	id, convErr := strconv.Atoi(ideaId)
	if convErr != nil {
		id = 0
	}
	if id > 0 {
		models.DeleteIdea(id)
	}

	ideaListHandler(w, r)
}

func createHandler(w http.ResponseWriter, r *http.Request) {
	promptEntry := r.FormValue("articleConcept")
	imgPrompt := r.FormValue("imagePrompt")
	imgUrl := r.FormValue("imageUrl")
	downloadImg := r.FormValue("downloadImage")
	generateImg := r.FormValue("generateImage")
	includeYt := r.FormValue("includeYt")
	ytUrl := r.FormValue("ytUrl")
	length := r.FormValue("articleLength")
	gpt4 := r.FormValue("useGpt4")
	ideaId := r.FormValue("ideaId")
	article := ""
	title := ""
	var imgBytes []byte

	iLen, convErr := strconv.Atoi(length)
	if convErr != nil {
		iLen = 500
	}
	post := Post{
		Image:       imgBytes,
		Prompt:      promptEntry,
		ImagePrompt: imgPrompt,
		Length:      iLen,
	}
	useGpt4 := false
	if gpt4 == "true" {
		useGpt4 = true
	}
	newImgPrompt := ""
	if promptEntry != "" {
		wpTmpl := template.Must(template.New("web-prompt").Parse(viper.GetString("config.prompt.web-prompt")))
		webPrompt := new(bytes.Buffer)
		wpErr := wpTmpl.Execute(webPrompt, post)
		if wpErr != nil {
			post.Error = "Error generating web prompt from template: " + wpErr.Error()
		}
		fmt.Println("Prompt is: ", webPrompt.String())
		articleResp, err := openai.GenerateArticle(useGpt4, webPrompt.String(), viper.GetString("config.prompt.system-prompt"))
		if err != nil {
			post.Error = "Error generating article from OpenAI API: " + err.Error()
		}
		article = articleResp
		titleResp, err := openai.GenerateTitle(false, article, viper.GetString("config.prompt.title-prompt"), viper.GetString("config.prompt.system-prompt"))
		if err != nil {
			post.Error = "Error generating title from OpenAI API: " + err.Error()
		}
		title = titleResp
		if includeYt == "true" && ytUrl != "" {
			article = article + "\n<p>[embed]" + ytUrl + "[/embed]</p>"
		}
		post.Content = article
		post.Title = title
	} else {
		post.Error = "Please input an article idea first."
	}

	if post.Error == "" && generateImg == "true" && imgPrompt != "" {
		fmt.Println("Img Prompt in is: ", imgPrompt)
		imgTmpl := template.Must(template.New("img-prompt").Parse(viper.GetString("config.prompt.img-prompt")))
		imgBuiltPrompt := new(bytes.Buffer)
		imgErr := imgTmpl.Execute(imgBuiltPrompt, post)
		if imgErr != nil {
			post.Error = "Error generating image: " + imgErr.Error()
		}
		newImgPrompt = imgBuiltPrompt.String()
		fmt.Println("Img Prompt out is: ", newImgPrompt)
		imgBytes = generateImage(newImgPrompt)
		post.Image = imgBytes
	} else if post.Error == "" && downloadImg == "true" && imgUrl != "" {
		response, err := http.Get(imgUrl)
		if err != nil {
			post.Error = "Error downloading image: " + err.Error()
		}
		defer func() {
			response.Body.Close()
		}()
		if response.StatusCode != 200 {
			post.Error = "Bad response code downloading image: " + strconv.Itoa(response.StatusCode)
		}
		imgBytes, err = io.ReadAll(response.Body)
		if err != nil {
			post.Error = "Error reading image bytes: " + err.Error()
		}
		post.Image = imgBytes
	}
	post.ImageB64 = base64.StdEncoding.EncodeToString(imgBytes)
	fmt.Println(len(imgBytes))
	postToWordpress(post)

	buf := &bytes.Buffer{}
	err := resultsTpl.Execute(buf, post)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	updId, convErr := strconv.Atoi(ideaId)
	if convErr != nil {
		updId = 0
	}
	if updId > 0 {
		models.SetIdeaWritten(updId)
	}
	_, wtErr := buf.WriteTo(w)
	if wtErr != nil {
		fmt.Println("Error rendering results html:" + wtErr.Error())
	}
}
