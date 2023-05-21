package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"golang/models"
	"golang/stablediffusion"
	"golang/util"
	"html/template"
	"net/http"
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
	Concept        string `json:"concept"`
}

type WriteData struct {
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

type ArticleListData struct {
	ErrorCode string           `json:"error-code"`
	Articles  []models.Article `json:"articles"`
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
type SettingsData struct {
	ErrorCode string
	Settings  map[string]models.Setting
	Upscalers map[string]stablediffusion.Upscaler
	Samplers  map[string]stablediffusion.Algorithm
}
type TemplatesData struct {
	ErrorCode string
	Templates map[string]models.Template
}
type IndexData struct {
	ErrorCode       string
	WordPressStatus bool
	OpenAiStatus    bool
	SdStatus        bool
	UnsplashStatus  bool
	Greeting        template.HTML
	Selfie          string
	Settings        map[string]models.Setting
	IdeaCount       int
	LastTestTime    string
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
var settingsTpl = template.Must(template.ParseFiles(tmplPath("settings.html"), tmplPath("base.html")))
var templatesTpl = template.Must(template.ParseFiles(tmplPath("templates.html"), tmplPath("base.html")))
var restartTpl = template.Must(template.ParseFiles(tmplPath("restart.html"), tmplPath("base.html")))
var articleListTpl = template.Must(template.ParseFiles(tmplPath("articleList.html"), tmplPath("base.html")))
var articleTpl = template.Must(template.ParseFiles(tmplPath("article.html"), tmplPath("base.html")))

func indexHandler(w http.ResponseWriter, _ *http.Request) {
	settings, err := models.GetSettings()
	if err != nil {
		util.Logger.Error().Err(err).Msg("Error getting settings")
	}
	selfieB64 := base64.StdEncoding.EncodeToString(Selfie)
	ideaCount := models.GetOpenIdeaCount()
	indexData := IndexData{
		ErrorCode:       "",
		WordPressStatus: WordPressStatus,
		OpenAiStatus:    OpenAiStatus,
		SdStatus:        SdStatus,
		UnsplashStatus:  UnsplashStatus,
		Greeting:        template.HTML(Greeting),
		Selfie:          selfieB64,
		Settings:        settings,
		IdeaCount:       ideaCount,
		LastTestTime:    LastTestTime,
	}

	buf := &bytes.Buffer{}
	renderErr := indexTpl.Execute(buf, indexData)
	if renderErr != nil {
		util.Logger.Error().Err(renderErr).Msg("Error executing template")
	}
	_, renderErr = buf.WriteTo(w)
	if renderErr != nil {
		util.Logger.Error().Err(renderErr).Msg("Error writing template to buffer")
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
		idea, err := models.GetIdeaById(ideaId)
		if err != nil {
			util.Logger.Error().Err(err).Msg("Error getting idea by id")
		}
		ideaText = idea.IdeaText
	}

	gpt4 := Settings["ENABLE_GPT4"]
	gpt4enabled := false
	if gpt4 == "true" {
		gpt4enabled = true
	}
	writeData := WriteData{
		ErrorCode:   "",
		GPT4Enabled: gpt4enabled,
		IdeaText:    ideaText,
		IdeaId:      ideaId,
	}
	buf := &bytes.Buffer{}
	renderErr := writeTpl.Execute(buf, writeData)
	if renderErr != nil {
		util.Logger.Error().Err(renderErr).Msg("Error executing template")
	}
	_, renderErr = buf.WriteTo(w)
	if renderErr != nil {
		util.Logger.Error().Err(renderErr).Msg("Error writing template to buffer")
	}
}

func ideaListHandler(w http.ResponseWriter, _ *http.Request) {
	ideas, err := models.GetOpenIdeas()
	if err != nil {
		util.Logger.Error().Err(err).Msg("Error getting open ideas")
	}
	planData := PlanData{
		ErrorCode: "",
		Ideas:     ideas,
		Series:    nil,
	}
	buf := &bytes.Buffer{}
	renderErr := ideaListTpl.Execute(buf, planData)
	if renderErr != nil {
		util.Logger.Error().Err(renderErr).Msg("Error executing template")
	}
	_, renderErr = buf.WriteTo(w)
	if renderErr != nil {
		util.Logger.Error().Err(renderErr).Msg("Error writing template to buffer")
	}

}

func seriesListHandler(w http.ResponseWriter, _ *http.Request) {
	series, err := models.GetSeries()
	if err != nil {
		util.Logger.Error().Err(err).Msg("Error getting series")
	}
	planData := PlanData{
		ErrorCode: "",
		Ideas:     nil,
		Series:    series,
	}
	buf := &bytes.Buffer{}
	renderErr := seriesListTpl.Execute(buf, planData)
	if renderErr != nil {
		util.Logger.Error().Err(renderErr).Msg("Error executing template")
	}
	_, renderErr = buf.WriteTo(w)
	if renderErr != nil {
		util.Logger.Error().Err(renderErr).Msg("Error writing template to buffer")
	}

}

func articleListHandler(w http.ResponseWriter, _ *http.Request) {
	articles, err := models.GetArticles()
	if err != nil {
		util.Logger.Error().Err(err).Msg("Error getting articles")
	}
	articleData := ArticleListData{
		ErrorCode: "",
		Articles:  articles,
	}
	buf := &bytes.Buffer{}
	renderErr := articleListTpl.Execute(buf, articleData)
	if renderErr != nil {
		util.Logger.Error().Err(renderErr).Msg("Error executing template")
	}
	_, renderErr = buf.WriteTo(w)
	if renderErr != nil {
		util.Logger.Error().Err(renderErr).Msg("Error writing template to buffer")
	}

}

type ArticleData struct {
	ErrorCode string
	Article   models.Article
	ArticleId string
	MediaUrl  string
}

func articleHandler(w http.ResponseWriter, r *http.Request) {
	articleId := r.FormValue("articleId")
	id, err := strconv.Atoi(articleId)
	if err != nil {
		id = 0
	}
	var article models.Article
	articleData := ArticleData{
		ErrorCode: "",
		ArticleId: articleId,
		MediaUrl:  "",
	}
	if id > 0 {
		article, err = models.GetArticleById(id)
		if err != nil {
			util.Logger.Error().Err(err).Msg("Error getting article by id")
		} else {
			articleData.Article = article
		}
	}
	if articleData.Article.MediaId > 0 {
		mediaUrl, err := getWordPressMediaUrlFromId(articleData.Article.MediaId)
		if err != nil {
			util.Logger.Error().Err(err).Msg("Error getting media url from id")
		} else {
			articleData.MediaUrl = mediaUrl
			util.Logger.Info().Str("mediaUrl", mediaUrl).Msg("Got media url")
		}
	}

	buf := &bytes.Buffer{}
	renderErr := articleTpl.Execute(buf, articleData)
	if renderErr != nil {
		util.Logger.Error().Err(renderErr).Msg("Error executing template")
	}
	_, renderErr = buf.WriteTo(w)
	if renderErr != nil {
		util.Logger.Error().Err(renderErr).Msg("Error writing template to buffer")
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
			util.Logger.Error().Err(err).Msg("Error getting idea by id")
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
	renderErr := ideaTpl.Execute(buf, ideaData)
	if renderErr != nil {
		util.Logger.Error().Err(renderErr).Msg("Error executing template")
	}
	_, renderErr = buf.WriteTo(w)
	if renderErr != nil {
		util.Logger.Error().Err(renderErr).Msg("Error writing template to buffer")
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
			util.Logger.Error().Err(err).Msg("Error getting series by id")
		}
		ideas, err := models.GetOpenSeriesIdeas(seriesId)
		if err != nil {
			util.Logger.Error().Err(err).Msg("Error getting ideas for series")
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
	renderErr := seriesTpl.Execute(buf, seriesData)
	if renderErr != nil {
		util.Logger.Error().Err(renderErr).Msg("Error executing template")
	}
	_, renderErr = buf.WriteTo(w)
	if renderErr != nil {
		util.Logger.Error().Err(renderErr).Msg("Error writing template to buffer")
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
		if err != nil {
			util.Logger.Error().Err(err).Msg("Error updating idea")
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
			util.Logger.Error().Err(err).Msg("Error adding idea")
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
			util.Logger.Error().Err(err).Msg("Error updating series")
		}
	} else {
		//Insert New
		series := models.Series{
			SeriesName:   seriesName,
			SeriesPrompt: seriesPrompt,
		}
		id, err := models.AddSeriesReturningId(series)
		if err != nil {
			util.Logger.Error().Err(err).Msg("Error adding series")
		}
		seriesId = strconv.Itoa(id)
	}

	series, err := models.GetSeriesById(seriesId)
	if err != nil {
		util.Logger.Error().Err(err).Msg("Error getting series")
	}
	ideas, err := models.GetOpenSeriesIdeas(seriesId)
	if err != nil {
		util.Logger.Error().Err(err).Msg("Error getting ideas")
	}
	seriesData := SeriesData{
		ErrorCode: "",
		Series:    series,
		Ideas:     ideas,
	}
	buf := &bytes.Buffer{}
	renderErr := seriesTpl.Execute(buf, seriesData)
	if renderErr != nil {
		util.Logger.Error().Err(renderErr).Msg("Error executing template")
	}
	_, renderErr = buf.WriteTo(w)
	if renderErr != nil {
		util.Logger.Error().Err(renderErr).Msg("Error writing template")
	}

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

	iId, err := strconv.Atoi(ideaId)
	if err != nil {
		iId = 0
	}
	concept := ""
	if iId > 0 {
		idea, err := models.GetIdeaById(ideaId)
		if err != nil {
			util.Logger.Error().Err(err).Msg("Error getting idea")
		}
		concept = idea.IdeaConcept
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
		Concept:        concept,
	}

	err, post = writeArticle(post)
	if err != nil {
		post.Error = err.Error()
	}
	buf := &bytes.Buffer{}
	renderErr := resultsTpl.Execute(buf, post)
	if renderErr != nil {
		util.Logger.Error().Err(renderErr).Msg("Error executing template")
	}
	_, renderErr = buf.WriteTo(w)
	if renderErr != nil {
		util.Logger.Error().Err(renderErr).Msg("Error writing template")
	}
}

func testHandler(w http.ResponseWriter, r *http.Request) {
	buf := &bytes.Buffer{}
	renderErr := resultsTpl.Execute(buf, nil)
	if renderErr != nil {
		util.Logger.Error().Err(renderErr).Msg("Error executing template")
	}
	_, renderErr = buf.WriteTo(w)
	if renderErr != nil {
		util.Logger.Error().Err(renderErr).Msg("Error writing template")
	}

}

func settingsHandler(w http.ResponseWriter, r *http.Request) {
	settings, err := models.GetSettings()

	if err != nil {
		util.Logger.Error().Err(err).Msg("Error getting settings")
	}
	sdUrl := Settings["SD_URL"]
	ctx := context.Background()
	upscalers, err := stablediffusion.GetUpscalers(sdUrl, ctx)
	if err != nil {
		util.Logger.Error().Err(err).Msg("Error getting upscalers")
	}
	samplers, err := stablediffusion.GetSamplers(sdUrl, ctx)
	if err != nil {
		util.Logger.Error().Err(err).Msg("Error getting samplers")
	}
	settingsData := SettingsData{
		ErrorCode: "",
		Settings:  settings,
		Upscalers: upscalers,
		Samplers:  samplers,
	}
	buf := &bytes.Buffer{}
	renderErr := settingsTpl.Execute(buf, settingsData)
	if renderErr != nil {
		util.Logger.Error().Err(renderErr).Msg("Error executing template")
	}
	_, renderErr = buf.WriteTo(w)
	if renderErr != nil {
		util.Logger.Error().Err(renderErr).Msg("Error writing template")
	}
}

func settingsSaveHandler(w http.ResponseWriter, r *http.Request) {
	settings := map[string]string{}
	_ = r.FormValue("BLOGOTRON_PORT")
	for k, v := range r.Form {
		settings[k] = v[0]
		_, err := models.UpsertSetting(k, v[0])
		if err != nil {
			util.Logger.Error().Err(err).Msg("Error updating setting")
		}
	}
	loadSettings()
	restartHandler(w, r)
}

func templateHandler(w http.ResponseWriter, r *http.Request) {
	templates, err := models.GetTemplates()
	templatesDate := TemplatesData{
		ErrorCode: "",
		Templates: templates,
	}
	if err != nil {
		util.Logger.Error().Err(err).Msg("Error getting templates")
	}

	buf := &bytes.Buffer{}
	renderErr := templatesTpl.Execute(buf, templatesDate)
	if renderErr != nil {
		util.Logger.Error().Err(renderErr).Msg("Error executing template")
	}
	_, renderErr = buf.WriteTo(w)
	if renderErr != nil {
		util.Logger.Error().Err(renderErr).Msg("Error writing template")
	}
}

func templateSaveHandler(w http.ResponseWriter, r *http.Request) {
	templates := map[string]string{}
	_ = r.FormValue("system-prompt")
	for k, v := range r.Form {
		templates[k] = v[0]
		_, err := models.UpsertTemplate(k, v[0])
		if err != nil {
			util.Logger.Error().Err(err).Msg("Error updating setting")
		}
	}
	loadTemplates()
	templateHandler(w, r)
}

func retestHandler(w http.ResponseWriter, r *http.Request) {
	if r.FormValue("test") == "wordpress" {
		testWordPress()
	} else if r.FormValue("test") == "openai" {
		testOpenAI()
	} else if r.FormValue("test") == "unsplash" {
		testUnsplash()
	} else if r.FormValue("test") == "sd" {
		testStableDiffusion()
	} else {
		runSystemTests()
	}
	indexHandler(w, r)
}

func restartHandler(w http.ResponseWriter, r *http.Request) {
	buf := &bytes.Buffer{}
	renderErr := restartTpl.Execute(buf, nil)
	if renderErr != nil {
		util.Logger.Error().Err(renderErr).Msg("Error executing template")
	}
	_, renderErr = buf.WriteTo(w)
	if renderErr != nil {
		util.Logger.Error().Err(renderErr).Msg("Error writing template")
	}
	util.Logger.Info().Msg("Restarting")
	task := Restart{}
	RestartChannel <- task
}
