package main

import (
	"bytes"
	"golang/models"
	"golang/util"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
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
	Keyword        string `json:"keyword"`
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

func tmplPath(file string) string {
	return filepath.Join("templates", file)
}

var resultsTpl = template.Must(template.ParseFiles(tmplPath("results.html"), tmplPath("base.html")))
var writeTpl = template.Must(template.ParseFiles(tmplPath("write.html"), tmplPath("base.html")))
var ideaListTpl = template.Must(template.ParseFiles(tmplPath("ideaList.html"), tmplPath("base.html")))
var seriesListTpl = template.Must(template.ParseFiles(tmplPath("seriesList.html"), tmplPath("base.html")))
var indexTpl = template.Must(template.ParseFiles(tmplPath("index.html"), tmplPath("base.html")))
var ideaTpl = template.Must(template.ParseFiles(tmplPath("idea.html"), tmplPath("base.html")))
var seriesTpl = template.Must(template.ParseFiles(tmplPath("series.html"), tmplPath("base.html")))

func indexHandler(w http.ResponseWriter, _ *http.Request) {
	indexData := PageData{
		ErrorCode: "",
	}
	buf := &bytes.Buffer{}
	err := indexTpl.Execute(buf, indexData)
	util.HandleError(err, "Error executing template")
	_, err = buf.WriteTo(w)
	util.HandleError(err, "Error writing template to buffer")
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
		util.HandleError(err, "Error getting idea by id")
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
	util.HandleError(err, "Error executing template")
	_, err = buf.WriteTo(w)
	util.HandleError(err, "Error writing template to buffer")
}

func ideaListHandler(w http.ResponseWriter, _ *http.Request) {
	ideas, err := models.GetOpenIdeas()
	util.HandleError(err, "Error getting open ideas")
	planData := PlanData{
		ErrorCode: "",
		Ideas:     ideas,
		Series:    nil,
	}
	buf := &bytes.Buffer{}
	err = ideaListTpl.Execute(buf, planData)
	util.HandleError(err, "Error executing template")
	_, err = buf.WriteTo(w)
	util.HandleError(err, "Error writing template to buffer")
}

func seriesListHandler(w http.ResponseWriter, _ *http.Request) {
	series, err := models.GetSeries()
	util.HandleError(err, "Error getting series")
	planData := PlanData{
		ErrorCode: "",
		Ideas:     nil,
		Series:    series,
	}
	buf := &bytes.Buffer{}
	err = seriesListTpl.Execute(buf, planData)
	util.HandleError(err, "Error executing template")
	_, err = buf.WriteTo(w)
	util.HandleError(err, "Error writing template to buffer")
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
		util.HandleError(err, "Error getting idea by id")
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
	util.HandleError(err, "Error executing template")
	_, err = buf.WriteTo(w)
	util.HandleError(err, "Error writing template to buffer")
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
		util.HandleError(err, "Error getting series by id")
		ideas, err := models.GetOpenSeriesIdeas(seriesId)
		util.HandleError(err, "Error getting ideas for series")
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
	util.HandleError(err, "Error executing template")
	_, err = buf.WriteTo(w)
	util.HandleError(err, "Error writing template to buffer")
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
		util.HandleError(err, "Error updating idea")
	} else {
		//Insert New
		idea := models.Idea{
			IdeaText: ideaText,
			Status:   "NEW",
			SeriesId: sid,
		}
		_, err := models.AddIdea(idea)
		util.HandleError(err, "Error adding idea")
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
		util.HandleError(err, "Error updating series")
	} else {
		//Insert New
		series := models.Series{
			SeriesName:   seriesName,
			SeriesPrompt: seriesPrompt,
		}
		id, err := models.AddSeriesReturningId(series)
		util.HandleError(err, "Error adding series")
		seriesId = strconv.Itoa(id)
	}

	series, err := models.GetSeriesById(seriesId)
	util.HandleError(err, "Error getting series")
	ideas, err := models.GetOpenSeriesIdeas(seriesId)
	util.HandleError(err, "Error getting ideas")
	seriesData := SeriesData{
		ErrorCode: "",
		Series:    series,
		Ideas:     ideas,
	}
	buf := &bytes.Buffer{}
	err = seriesTpl.Execute(buf, seriesData)
	util.HandleError(err, "Error executing template")
	_, err = buf.WriteTo(w)
	util.HandleError(err, "Error writing template")
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

	err, post := writeArticle(post)
	if err != nil {
		post.Error = err.Error()
	}
	buf := &bytes.Buffer{}
	err = resultsTpl.Execute(buf, post)
	util.HandleError(err, "Error executing template")
	_, err = buf.WriteTo(w)
	util.HandleError(err, "Error writing template")
}

func testHandler(w http.ResponseWriter, r *http.Request) {
	buf := &bytes.Buffer{}
	renderErr := resultsTpl.Execute(buf, nil)
	util.HandleError(renderErr, "Error executing template")
	_, renderErr = buf.WriteTo(w)
	util.HandleError(renderErr, "Error writing template")
}
