package main

import (
	"bytes"
	"encoding/base64"
	"golang/models"
	"golang/unsplash"
	"html/template"
	"net/http"
	"os"
	"strconv"
	"strings"
)

type Post struct {
	Title          string `json:"title"`
	Content        string `json:"content"`
	Description    string `json:"description"`
	Image          []byte `json:"image"`
	Prompt         string `json:"prompt"`
	ImagePrompt    string `json:"image-prompt"`
	Error          string `json:"error"`
	ImageB64       string `json:"image64"`
	Length         int    `json:"article-length"`
	PublishStatus  string `json:"publish-status"`
	UseGpt4        bool   `json:"use-gpt4"`
	ConceptAsTitle bool   `json:"concept-as-title"`
	IncludeYt      bool   `json:"include-yt"`
	YtUrl          string `json:"yt-url"`
	GenerateImg    bool   `json:"generate-img"`
	DownloadImg    bool   `json:"download-img"`
	ImgUrl         string `json:"img-url"`
	UnsplashImg    bool   `json:"unsplash-img"`
	IdeaId         string `json:"idea-id"`
	UnsplashSearch string `json:"unsplash-search"`
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
	handleError(err)
	_, err = buf.WriteTo(w)
	handleError(err)
}

func writeHandler(w http.ResponseWriter, r *http.Request) {
	ideaId := r.FormValue("ideaId")
	id, err := strconv.Atoi(ideaId)
	if err != nil {
		id = 0
	}
	ideaText := ""
	if id > 0 {
		idea, err := models.GetIdeaById(ideaId)
		handleError(err)
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
	handleError(err)
	_, err = buf.WriteTo(w)
	handleError(err)
}

func ideaListHandler(w http.ResponseWriter, _ *http.Request) {
	ideas, err := models.GetOpenIdeas()
	handleError(err)
	planData := PlanData{
		ErrorCode: "",
		Ideas:     ideas,
		Series:    nil,
	}
	buf := &bytes.Buffer{}
	err = ideaListTpl.Execute(buf, planData)
	handleError(err)
	_, err = buf.WriteTo(w)
	handleError(err)
}

func seriesListHandler(w http.ResponseWriter, _ *http.Request) {
	series, err := models.GetSeries()
	handleError(err)
	planData := PlanData{
		ErrorCode: "",
		Ideas:     nil,
		Series:    series,
	}
	buf := &bytes.Buffer{}
	err = seriesListTpl.Execute(buf, planData)
	handleError(err)
	_, err = buf.WriteTo(w)
	handleError(err)
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
		handleError(err)
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
	handleError(err)
	_, err = buf.WriteTo(w)
	handleError(err)
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
		handleError(err)
		ideas, err := models.GetOpenSeriesIdeas(seriesId)
		handleError(err)
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
	handleError(err)
	_, err = buf.WriteTo(w)
	handleError(err)
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
		seriesId = "0"
	}
	ideaCount := r.FormValue("ideaCount")
	if strings.TrimSpace(ideaCount) == "" {
		ideaCount = "10"
	}
	ideaConcept := r.FormValue("ideaConcept")
	builtConcept := ideaConcept
	builtFresh := false
	if sid > 0 {
		series, _ := models.GetSeriesById(seriesId)
		if strings.TrimSpace(series.SeriesPrompt) != "" {
			ideaList := ""
			builtConcept = "The topic for the ideas is: \"" + series.SeriesPrompt + "\"."
			//Get ideas for this series and iterate them adding existing ideas to a list of ideas
			ideas, _ := models.GetSeriesIdeas(seriesId)
			for _, idea := range ideas {
				ideaList = ideaList + ", " + idea.IdeaText
			}
			if strings.TrimSpace(ideaList) != "" {
				builtConcept = builtConcept + " The following ideas have already been used: " + ideaList + "."
			}
		}
	} else {
		ideaList := ""
		if strings.TrimSpace(ideaConcept) != "" {
			builtConcept = "The topic for the ideas is: \"" + ideaConcept + "\"."
			ideas, _ := models.GetIdeasByConcept(ideaConcept)
			for _, idea := range ideas {
				ideaList = ideaList + ", " + idea.IdeaText
			}
			if strings.TrimSpace(ideaList) != "" {
				builtConcept = builtConcept + " The following ideas have already been used: " + ideaList + "."
			}
		} else {
			fullBrainstorm(ideaCount, useGpt4)
			builtFresh = true
		}

	}
	if !builtFresh {
		generateIdeas(ideaCount, builtConcept, useGpt4, sid, ideaConcept)
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
		handleError(err)
	} else {
		//Insert New
		idea := models.Idea{
			IdeaText: ideaText,
			Status:   "NEW",
			SeriesId: sid,
		}
		_, err := models.AddIdea(idea)
		handleError(err)
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
		handleError(err)
	} else {
		//Insert New
		series := models.Series{
			SeriesName:   seriesName,
			SeriesPrompt: seriesPrompt,
		}
		id, err := models.AddSeriesReturningId(series)
		handleError(err)
		seriesId = strconv.Itoa(id)
	}

	series, err := models.GetSeriesById(seriesId)
	handleError(err)
	ideas, err := models.GetOpenSeriesIdeas(seriesId)
	handleError(err)
	seriesData := SeriesData{
		ErrorCode: "",
		Series:    series,
		Ideas:     ideas,
	}
	buf := &bytes.Buffer{}
	err = seriesTpl.Execute(buf, seriesData)
	handleError(err)
	_, err = buf.WriteTo(w)
	handleError(err)
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
	unsplashImg := r.FormValue("unsplashImage")
	unsplashSearch := r.FormValue("unsplashPrompt")
	publishStatus := r.FormValue("publishStatus")
	conceptAsTitle := r.FormValue("conceptAsTitle")
	var imgBytes []byte

	iLen, convErr := strconv.Atoi(length)
	if convErr != nil {
		iLen = 500
	}

	post := Post{
		Title:          "",
		Content:        "",
		Description:    "",
		Image:          imgBytes,
		Prompt:         promptEntry,
		ImagePrompt:    imgPrompt,
		Error:          "",
		ImageB64:       "",
		Length:         iLen,
		PublishStatus:  publishStatus,
		UseGpt4:        gpt4 == "true",
		ConceptAsTitle: conceptAsTitle == "true",
		IncludeYt:      includeYt == "true",
		YtUrl:          ytUrl,
		GenerateImg:    generateImg == "true",
		DownloadImg:    downloadImg == "true",
		ImgUrl:         imgUrl,
		UnsplashImg:    unsplashImg == "true",
		IdeaId:         ideaId,
		UnsplashSearch: unsplashSearch,
	}

	post = writeArticle(post)
	buf := &bytes.Buffer{}
	err := resultsTpl.Execute(buf, post)
	handleError(err)
	_, err = buf.WriteTo(w)
	handleError(err)
}

func testHandler(w http.ResponseWriter, r *http.Request) {
	//imgBytes := unsplash.GetRandomImage()
	imgBytes := unsplash.GetImageBySearch("cat")
	imgPrompt := "Testing unsplash random image"
	promptEntry := "Testing unsplash random image"
	post := Post{
		Image:       imgBytes,
		Prompt:      promptEntry,
		ImagePrompt: imgPrompt,
	}
	post.ImageB64 = base64.StdEncoding.EncodeToString(imgBytes)
	buf := &bytes.Buffer{}
	err := resultsTpl.Execute(buf, post)
	handleError(err)
	_, err = buf.WriteTo(w)
	handleError(err)
}
