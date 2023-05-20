package main

import (
	"bytes"
	"context"
	"embed"
	"encoding/base64"
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/go-co-op/gocron"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"golang/api"
	"golang/models"
	"golang/openai"
	"golang/stablediffusion"
	"golang/unsplash"
	"golang/util"
	"html/template"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

//go:embed sql/migrations/*.sql
var MigrationSrc embed.FS

var (
	Settings        map[string]string
	Templates       map[string]string
	WordPressStatus = false
	OpenAiStatus    = false
	SdStatus        = false
	UnsplashStatus  = false
	Greeting        string
	Selfie          []byte
	LastTestTime    string
)

func main() {
	util.Init()
	util.Logger.Info().Msg("Starting Blogotron")

	//DB
	dbName := "blogtron.db"
	err := models.ConnectDatabase(dbName)
	if err != nil {
		util.Logger.Error().Err(err).Msg("Could not connect to database file " + dbName)
		return
	}
	migSrc, err := iofs.New(MigrationSrc, "sql/migrations")
	if err != nil {
		util.Logger.Error().Err(err).Msg("Could not load migrations")
		return
	}
	err = models.MigrateDatabase(migSrc)
	if err != nil {
		util.Logger.Error().Err(err).Msg("Could not migrate database")
		return
	}
	Settings, err = models.GetSettingsSimple()
	if err != nil {
		util.Logger.Error().Err(err).Msg("Could not load settings from db")
		return
	}
	Templates, err = models.GetTemplatesSimple()
	if err != nil {
		util.Logger.Error().Err(err).Msg("Could not load templates from db")
		return
	}

	if Settings["ENABLE_STARTUP_TESTS"] == "true" {
		runSystemTests()
	} else {
		loadCachedTestResults()
	}

	apiPort := Settings["BLOGOTRON_API_PORT"]
	apiGin := gin.Default()
	v1 := apiGin.Group("/api/v1")
	{
		v1.GET("idea", api.GetIdeas)
		v1.GET("idea/:id", api.GetIdeaById)
		v1.POST("idea", api.AddIdea)
		v1.PUT("idea/:id", api.UpdateIdea)
		v1.DELETE("idea/:id", api.DeleteIdea)
		v1.OPTIONS("idea", api.Options)
	}

	//WEB SERVER
	webPort := Settings["BLOGOTRON_PORT"]
	if webPort == "" {
		webPort = "8666"
	}
	fs := http.FileServer(http.Dir("assets"))
	mux := http.NewServeMux()

	mux.HandleFunc("/", indexHandler)
	mux.HandleFunc("/create", createHandler)
	mux.HandleFunc("/write", writeHandler)
	mux.HandleFunc("/ideaList", ideaListHandler)
	mux.HandleFunc("/aiIdea", aiIdeaHandler)
	mux.HandleFunc("/idea", ideaHandler)
	mux.HandleFunc("/ideaSave", ideaSaveHandler)
	mux.HandleFunc("/ideaDel", ideaRemoveHandler)
	mux.HandleFunc("/series", seriesHandler)
	mux.HandleFunc("/seriesList", seriesListHandler)
	mux.HandleFunc("/seriesSave", seriesSaveHandler)
	mux.HandleFunc("/settings", settingsHandler)
	mux.HandleFunc("/settingsSave", settingsSaveHandler)
	mux.HandleFunc("/templates", templateHandler)
	mux.HandleFunc("/templatesSave", templateSaveHandler)
	mux.HandleFunc("/test", testHandler)
	mux.HandleFunc("/retest", retestHandler)
	mux.Handle("/assets/", http.StripPrefix("/assets/", fs))

	//Cron Service
	autoPost := Settings["AUTO_POST_ENABLE"]
	autoPostInterval := Settings["AUTO_POST_INTERVAL"]
	autoPostImgEngine := Settings["AUTO_POST_IMG_ENGINE"]
	autoPostLen := Settings["AUTO_POST_LEN"]
	autoPostState := Settings["AUTO_POST_STATE"]
	lowIdeaThreshold := Settings["LOW_IDEA_THRESHOLD"]
	cronSrv := gocron.NewScheduler(time.UTC)
	iThreshold, convErr := strconv.Atoi(lowIdeaThreshold)
	if convErr != nil {
		iThreshold = 0
	}
	if iThreshold > 0 {
		util.Logger.Info().Msg("Auto Idea Generation Enabled - Low Idea Threshold Set to " + strconv.Itoa(iThreshold) + " ideas")
		cronSrv.Every("1h").Do(func() {
			util.Logger.Info().Msg("Checking Idea Levels")
			//Check count of total open ideas
			ideaCount := models.GetOpenIdeaCount()
			util.Logger.Info().Msg("Idea Count: " + strconv.Itoa(ideaCount) + " Threshold: " + strconv.Itoa(iThreshold))
			if ideaCount < iThreshold {
				util.Logger.Info().Msg("Idea Count is below threshold - Generating 10 new concepts and 10 new ideas for each concept")
				fullBrainstorm("10", false)
			}
		})
	} else {
		util.Logger.Info().Msg("Auto Idea Generation Disabled")
	}
	if autoPost == "true" {
		util.Logger.Info().Msg("Auto Post Enabled - Interval Set to " + autoPostInterval)
		cronSrv.Every(autoPostInterval).Do(func() {
			util.Logger.Info().Msg("Auto Post Triggered")
			//Get a Random Idea
			idea := models.GetRandomIdea()
			if idea.Id == 0 {
				util.Logger.Info().Msg("Could not get random idea")
			} else {
				util.Logger.Info().Msg("Random Idea: " + idea.IdeaText)
				//Create a new post from the idea
				iLen, convErr := strconv.Atoi(autoPostLen)
				if convErr != nil {
					iLen = 750
				}
				publishStatus := autoPostState
				unsplashSearch := ""
				unsplashImg := false
				generateImg := false
				if autoPostImgEngine == "unsplash" {
					unsplashImg = true
				} else if autoPostImgEngine == "generate" {
					generateImg = true
				}

				post := Post{
					Prompt:         idea.IdeaText,
					Length:         iLen,
					PublishStatus:  publishStatus,
					UseGpt4:        false,
					ConceptAsTitle: false,
					IncludeYt:      false,
					GenerateImg:    generateImg,
					DownloadImg:    false,
					UnsplashImg:    unsplashImg,
					IdeaId:         strconv.Itoa(idea.Id),
					UnsplashSearch: unsplashSearch,
					Concept:        idea.IdeaConcept,
				}

				err, post = writeArticle(post)
				if err != nil {
					util.Logger.Error().Err(err).Msg("Could not write article")
				}
			}
		})
	}

	//Thread Mgmt
	util.Logger.Info().Msg("Starting Thread Management")
	wg := new(sync.WaitGroup)
	wg.Add(3)
	util.Logger.Info().Msg("Starting Gin Server")
	go func() {
		apiGin.Run(":" + apiPort)
		wg.Done()
	}()
	util.Logger.Info().Msg("Started Gin Server on Port " + apiPort + "")

	util.Logger.Info().Msg("Starting Web Server")
	go func() {
		err = http.ListenAndServe(":"+webPort, mux)
		if err != nil {
			util.Logger.Error().Err(err).Msg("Error starting web server")
		}
		wg.Done()
	}()
	util.Logger.Info().Msg("Started Web Server on Port " + webPort + "")

	util.Logger.Info().Msg("Starting Cron Server")
	go func() {
		//Start cron
		cronSrv.StartBlocking()
		wg.Done()
	}()
	util.Logger.Info().Msg("Started Cron Server")
	util.Logger.Info().Msg("Startup Completed")
	wg.Wait()
}

type SimilarityResponse struct {
	Title         string               `json:"title"`
	Score         string               `json:"score"`
	SimilarTitles []SimilarityResponse `json:"similar_titles"`
}

func generateSizedImage(p string, iWidth int, iHeight int) ([]byte, error) {
	var imgBytes []byte
	imgSampler := Settings["IMG_SAMPLER"]
	imgUpscaler := Settings["IMG_UPSCALER"]
	imgNegativePrompts := Settings["IMG_NEGATIVE_PROMPTS"]
	imgSteps := Settings["IMG_STEPS"]
	iSteps, err := strconv.Atoi(imgSteps)
	if err != nil {
		iSteps = 30
	}

	if p != "" {
		imgMode := Settings["IMG_MODE"]
		if imgMode == "openai" {
			aiApiKey := Settings["OPENAI_API_KEY"]
			imgBytes, err = openai.GenerateImg(p, aiApiKey)
			if err != nil {
				return nil, err
			}
		} else if imgMode == "sd" {
			sdUrl := Settings["SD_URL"]
			ctx := context.Background()
			images, err := stablediffusion.Generate(sdUrl, ctx, stablediffusion.SimpleImageRequest{
				Prompt:                            p,
				NegativePrompt:                    imgNegativePrompts,
				Styles:                            nil,
				Seed:                              -1,
				SamplerName:                       imgSampler,
				BatchSize:                         1,
				NIter:                             1,
				Steps:                             iSteps,
				CfgScale:                          7,
				Width:                             iWidth,
				Height:                            iHeight,
				SNoise:                            0,
				OverrideSettings:                  struct{}{},
				OverrideSettingsRestoreAfterwards: false,
				SaveImages:                        true,
				EnableHr:                          true,
				HrScale:                           2,
				HrUpscaler:                        imgUpscaler,
			})
			if err != nil {
				return nil, err
			} else {
				imgBytes = images.Images[0]
			}
		}
	}
	return imgBytes, nil
}

func generateImage(p string) ([]byte, error) {

	imgWidth := Settings["IMG_WIDTH"]
	imgHeight := Settings["IMG_HEIGHT"]

	iWidth, err := strconv.Atoi(imgWidth)
	if err != nil {
		iWidth = 512
	}
	iHeight, err := strconv.Atoi(imgHeight)
	if err != nil {
		iHeight = 512
	}
	return generateSizedImage(p, iWidth, iHeight)
}

func writeArticle(post Post) (error, Post) {
	newImgPrompt := ""
	article := ""
	title := ""
	aiApiKey := Settings["OPENAI_API_KEY"]
	if post.Prompt != "" {
		if post.Keyword == "" {
			kwTmpl := template.Must(template.New("keyword-prompt").Parse(openai.KeywordTemplate))
			keywordPrompt := new(bytes.Buffer)
			err := kwTmpl.Execute(keywordPrompt, post)
			if err != nil {
				return err, post
			}
			keywordResp, err := openai.GenerateKeywords(aiApiKey, post.UseGpt4, keywordPrompt.String(), Templates["system-prompt"])
			if err != nil {
				return err, post
			}
			post.Keyword = keywordResp
		}
		wpTmpl := template.Must(template.New("web-prompt").Parse(Templates["article-prompt"]))
		webPrompt := new(bytes.Buffer)
		err := wpTmpl.Execute(webPrompt, post)
		if err != nil {
			return err, post
		}
		util.Logger.Info().Msg("Generating Article from Prompt" + webPrompt.String() + "")
		articleResp, err := openai.GenerateArticle(aiApiKey, post.UseGpt4, webPrompt.String(), Templates["system-prompt"])
		if err != nil {
			return err, post
		}
		article = articleResp
		//Attempt to parse out title from h1 tag
		if strings.Contains(article, "<h1>") && strings.Contains(article, "</h1>") && strings.HasPrefix(article, "<h1>") {
			tempTitle := strings.Split(strings.Split(article, "<h1>")[1], "</h1>")[0]
			if tempTitle != "Introduction" {
				title = tempTitle
				//Remove title from article
				article = strings.Replace(article, "<h1>"+title+"</h1>", "", 1)
				//Remove any leading newlines from article
				article = strings.TrimPrefix(article, "\n")
			}
		}
		if title == "" {
			if !post.ConceptAsTitle {
				titleResp, err := openai.GenerateTitle(aiApiKey, false, article, Templates["title-prompt"], Templates["system-prompt"])
				if err != nil {
					return err, post
				}
				title = titleResp
			} else {
				title = post.Prompt
			}
		}
		//Generate description
		if post.Description == "" {
			descTmpl := template.Must(template.New("description-prompt").Parse(Templates["description-prompt"]))
			descPrompt := new(bytes.Buffer)
			err := descTmpl.Execute(descPrompt, post)
			descResp, err := openai.GenerateDescription(aiApiKey, false, article, descPrompt.String(), Templates["system-prompt"])
			if err != nil {
				return err, post
			}
			post.Description = descResp
		}
		if post.IncludeYt && post.YtUrl != "" {
			article = article + "\n<p>[embed]" + post.YtUrl + "[/embed]</p>"
		}
		//if title starts with a quote, remove it and if title ends with a quote, remove it
		if strings.HasPrefix(title, "\"") {
			title = strings.TrimPrefix(title, "\"")
		}
		if strings.HasSuffix(title, "\"") {
			title = strings.TrimSuffix(title, "\"")
		}
		post.Content = article
		post.Title = title
	} else {
		post.Error = "Please input an article idea first."
	}

	if post.Error == "" && post.GenerateImg {
		if post.ImagePrompt == "" {
			igTmpl := template.Must(template.New("imggen-prompt").Parse(Templates["imggen-prompt"]))
			imgGenPrompt := new(bytes.Buffer)
			err := igTmpl.Execute(imgGenPrompt, post)
			imgGenResp, err := openai.GenerateImagePrompt(aiApiKey, false, title, imgGenPrompt.String(), Templates["system-prompt"])
			if err != nil {
				return err, post
			}
			imgGenResp = strings.Replace(imgGenResp, "\"", "", 1)
			imgGenResp = strings.Replace(imgGenResp, "Create an image of ", "", 1)
			imgGenResp = strings.Replace(imgGenResp, "Can you create an image of ", "", 1)
			post.ImagePrompt = imgGenResp
		}
		util.Logger.Info().Msg("Img Prompt in is: " + post.ImagePrompt)
		imgTmpl := template.Must(template.New("img-prompt").Parse(Templates["img-prompt"]))
		imgBuiltPrompt := new(bytes.Buffer)
		err := imgTmpl.Execute(imgBuiltPrompt, post)
		if err != nil {
			return err, post
		}
		newImgPrompt = imgBuiltPrompt.String()
		util.Logger.Info().Msg("Img Prompt Out is: " + newImgPrompt)
		imgBytes, err := generateImage(newImgPrompt)
		if err != nil {
			return err, post
		}
		post.Image = imgBytes
	} else if post.Error == "" && post.DownloadImg && post.ImgUrl != "" {
		response, err := http.Get(post.ImgUrl)
		if err != nil {
			return err, post
		}
		defer func() {
			response.Body.Close()
		}()
		if response.StatusCode != 200 {
			post.Error = "Bad response code downloading image: " + strconv.Itoa(response.StatusCode)
		}
		imgBytes, err := io.ReadAll(response.Body)
		if err != nil {
			return err, post
		}
		post.Image = imgBytes
	} else if post.Error == "" && post.UnsplashImg && post.UnsplashSearch != "" {
		unsplashKey := Settings["UNSPLASH_ACCESS_KEY"]
		imgBytes, err := unsplash.GetImageBySearch(unsplashKey, post.UnsplashSearch)
		if err != nil {
			return err, post
		}
		post.Image = imgBytes
	} else if post.Error == "" && post.UnsplashImg && post.UnsplashSearch == "" {
		imgSearchResp, err := openai.GenerateImageSearch(aiApiKey, false, title, Templates["imgsearch-prompt"], Templates["system-prompt"])
		if err != nil {
			return err, post
		}
		post.UnsplashSearch = imgSearchResp
		unsplashKey := Settings["UNSPLASH_ACCESS_KEY"]
		imgBytes, err := unsplash.GetImageBySearch(unsplashKey, imgSearchResp)
		if err != nil {
			return err, post
		}
		post.Image = imgBytes
	}
	post.ImageB64 = base64.StdEncoding.EncodeToString(post.Image)
	err := postToWordpress(post)
	if err != nil {
		return err, post
	} else {
		models.SetIdeaWritten(post.IdeaId)
	}
	return nil, post
}

func generateIdeas(ideaCount string, builtConcept string, useGpt4 bool, sid int, ideaConcept string) {
	prompt := Prompt{
		IdeaCount:   ideaCount,
		IdeaConcept: builtConcept,
	}
	aiApiKey := Settings["OPENAI_API_KEY"]
	ideaTmpl := template.Must(template.New("idea-prompt").Parse(openai.IdeaTemplate))
	ideaPrompt := new(bytes.Buffer)
	err := ideaTmpl.Execute(ideaPrompt, prompt)
	if err != nil {
		util.Logger.Error().Err(err).Msg("Error executing idea template")
	} else {
		util.Logger.Info().Msg("Prompt is: " + ideaPrompt.String())
		ideaResp, err := openai.GenerateIdeas(aiApiKey, useGpt4, ideaPrompt.String(), Templates["system-prompt"])
		if err != nil {
			util.Logger.Error().Err(err).Msg("Error generating ideas")
		} else {
			ideaResp = strings.ReplaceAll(ideaResp, "\n", "")
			util.Logger.Info().Msg("Idea Brainstorm Results: " + ideaResp)
			ideaList := strings.Split(ideaResp, "|")
			for index, value := range ideaList {
				util.Logger.Info().Msgf("Index: %d, Value: %s\n", index, value)
				if strings.TrimSpace(value) != "" {
					idea := models.Idea{
						IdeaText:    strings.TrimSpace(value),
						Status:      "NEW",
						IdeaConcept: ideaConcept,
						SeriesId:    sid,
					}
					_, err = models.AddIdea(idea)
					if err != nil {
						util.Logger.Error().Err(err).Msg("Error adding idea")
					}
				}
			}
		}
	}

}

func fullBrainstorm(ideaCount string, useGpt4 bool) {
	conceptList := ""
	builtTopic := ""
	aiApiKey := Settings["OPENAI_API_KEY"]
	concepts, _ := models.GetIdeaConcepts()
	series, _ := models.GetSeries()
	for _, concept := range concepts {
		conceptList = conceptList + ", " + concept
	}
	for _, s := range series {
		conceptList = conceptList + ", " + s.SeriesPrompt
	}
	builtTopic = builtTopic + " Previous topics used include: " + conceptList + "."
	ideaTmpl := template.Must(template.New("topic-prompt").Parse(openai.TopicTemplate))
	ideaPrompt := new(bytes.Buffer)
	prompt := Prompt{
		IdeaCount:   ideaCount,
		IdeaConcept: builtTopic,
	}
	err := ideaTmpl.Execute(ideaPrompt, prompt)
	if err != nil {
		util.Logger.Error().Err(err).Msg("Error executing idea template")
	} else {
		util.Logger.Info().Msg("Topic Prompt is: " + ideaPrompt.String())
		ideaResp, err := openai.GenerateTopics(aiApiKey, useGpt4, ideaPrompt.String(), Templates["system-prompt"])
		if err != nil {
			util.Logger.Error().Err(err).Msg("Error generating ideas")
		} else {
			ideaResp = strings.ReplaceAll(ideaResp, "\n", "")
			util.Logger.Info().Msg("Idea Brainstorm Results: " + ideaResp)
			ideaList := strings.Split(ideaResp, "|")
			for _, value := range ideaList {
				builtConcept := "The topic for the ideas is: \"" + value + "\"."
				generateIdeas(ideaCount, builtConcept, useGpt4, 0, value)
			}
		}
	}
}

func getWpTitles() ([]string, error) {
	// Create an HTTP client
	client := &http.Client{}

	// Define the URL and request method
	baseUrl := Settings["WP_URL"]
	//IF baseUrl ends with a slash, remove it
	if baseUrl[len(baseUrl)-1:] == "/" {
		baseUrl = baseUrl[:len(baseUrl)-1]
	}
	url := baseUrl + "/wp-json/wp/v2/posts?per_page=100"
	method := "GET"

	// Define the authentication credentials
	username := Settings["WP_USERNAME"]
	password := Settings["WP_PASSWORD"]

	// Create a request
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		util.Logger.Error().Err(err).Msg("Error creating request for title list")
		return nil, err
	}

	// Set the authentication header
	req.SetBasicAuth(username, password)

	// Send the request
	res, err := client.Do(req)
	if err != nil {
		util.Logger.Error().Err(err).Msg("Error sending request for title list")
		return nil, err
	}

	// Read the response body
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		util.Logger.Error().Err(err).Msg("Error reading response body for title list")
		return nil, err
	}

	// Close the response body
	res.Body.Close()

	// Parse the JSON data
	var posts []map[string]interface{}
	err = json.Unmarshal(body, &posts)
	if err != nil {
		util.Logger.Error().Err(err).Msg("Error parsing JSON for title list")
		return nil, err
	}

	// Create a slice of strings to store the titles
	var titles []string

	// Loop through the posts and add the titles to the slice
	for _, post := range posts {
		titles = append(titles, post["title"].(map[string]interface{})["rendered"].(string))
	}

	// Return the titles
	return titles, nil
}

func doWordpressPost(endPoint string, postData map[string]interface{}) error {
	// Convert the post data to JSON
	jsonData, err := json.Marshal(postData)
	if err != nil {
		return err
	}

	// Create an HTTP client
	client := &http.Client{}

	// Define the URL and request method
	baseUrl := Settings["WP_URL"]
	//IF baseUrl ends with a slash, remove it
	if baseUrl[len(baseUrl)-1:] == "/" {
		baseUrl = baseUrl[:len(baseUrl)-1]
	}
	url := baseUrl + endPoint //"/wp-json/wp/v2/posts"
	method := "POST"

	// Define the authentication credentials
	username := Settings["WP_USERNAME"]
	password := Settings["WP_PASSWORD"]

	// Create a request with the JSON data
	req, err := http.NewRequest(method, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	// Calculate the content length
	contentLength := strconv.Itoa(len(jsonData))

	// Set the content type header
	req.Header.Set("Content-Type", "application/json")
	// Set the content length header
	req.Header.Set("Content-Length", contentLength)
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Connection", "keep-alive")
	// Set the host header
	req.Header.Set("Host", strings.ReplaceAll(strings.ReplaceAll(Settings["WP_URL"], "https://", ""), "http://", ""))
	req.Header.Set("User-Agent", "PostmanRuntime/7.26.8")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")

	// Encode the username and password in base64
	auth := username + ":" + password
	basicAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte(auth))

	// Set the Authorization header for basic authentication
	req.Header.Set("Authorization", basicAuth)

	// Send the request
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	// Check the response status code
	if res.StatusCode != http.StatusCreated {
		return errors.New("Post creation failed. Status code:" + strconv.Itoa(res.StatusCode))
	}
	return nil
}

type MediaResponse struct {
	ID int `json:"id"`
}

func postImageToWordpress(imgBytes []byte, description string) int {
	// Create a new multipart writer
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Create a new form file field for the image
	part, err := writer.CreateFormFile("file", "image.jpg")
	if err != nil {
		util.Logger.Error().Err(err).Msg("Error creating form file field")
		return 0
	}

	// Copy the image bytes to the form file field
	_, err = io.Copy(part, bytes.NewReader(imgBytes))
	if err != nil {
		util.Logger.Error().Err(err).Msg("Error copying image bytes")
		return 0
	}

	// Add the alt text as a form field
	err = writer.WriteField("alt_text", description)
	if err != nil {
		util.Logger.Error().Err(err).Msg("Error writing alt text field")
	}

	// Close the multipart writer
	err = writer.Close()
	if err != nil {
		util.Logger.Error().Err(err).Msg("Error closing multipart writer")
		return 0
	}
	// Create an HTTP client
	client := &http.Client{}

	// Define the URL and request method
	baseUrl := Settings["WP_URL"]
	//IF baseUrl ends with a slash, remove it
	if baseUrl[len(baseUrl)-1:] == "/" {
		baseUrl = baseUrl[:len(baseUrl)-1]
	}
	url := baseUrl + "/wp-json/wp/v2/media"
	method := "POST"

	// Define the authentication credentials
	username := Settings["WP_USERNAME"]
	password := Settings["WP_PASSWORD"]

	// Create a request with the multipart body
	req, err := http.NewRequest(method, url, body)
	mediaID := 0
	if err != nil {
		util.Logger.Error().Err(err).Msg("Error creating request for image upload")
		return 0
	} else {
		// Calculate the content length
		contentLength := strconv.Itoa(body.Len())

		// Set the content type header
		req.Header.Set("Content-Type", writer.FormDataContentType())
		// Set the content length header
		req.Header.Set("Content-Length", contentLength)
		req.Header.Set("Cache-Control", "no-cache")
		req.Header.Set("Accept", "*/*")
		req.Header.Set("Connection", "keep-alive")
		// Set the host header
		req.Header.Set("Host", strings.ReplaceAll(strings.ReplaceAll(baseUrl, "https://", ""), "http://", ""))
		req.Header.Set("User-Agent", "PostmanRuntime/7.26.8")
		req.Header.Set("Accept-Encoding", "gzip, deflate, br")

		// Encode the username and password in base64
		auth := username + ":" + password
		basicAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte(auth))

		// Set the Authorization header for basic authentication
		req.Header.Set("Authorization", basicAuth)
		// Send the request
		res, err := client.Do(req)
		if err != nil {
			util.Logger.Error().Err(err).Msg("Error sending request")
			return 0
		}
		defer res.Body.Close()

		// Read the response body
		responseBody, err := ioutil.ReadAll(res.Body)
		if err != nil {
			util.Logger.Error().Err(err).Msg("Error reading response body")
			return 0
		}

		// Check the response status code
		if res.StatusCode != http.StatusCreated {
			util.Logger.Error().Msg("Image upload failed with status code " + strconv.Itoa(res.StatusCode) + ". Response body: " + string(responseBody) + ".")
			return 0
		}

		// Parse the response body to get the media ID
		var mediaResp MediaResponse
		err = json.Unmarshal(responseBody, &mediaResp)
		if err != nil {
			util.Logger.Error().Err(err).Msg("Error parsing response body")
			return 0
		}

		mediaID = mediaResp.ID
		util.Logger.Info().Msg("Image uploaded successfully! Media ID:" + strconv.Itoa(mediaID))
	}

	return mediaID
}

func postToWordpress(post Post) error {
	postData := map[string]interface{}{}
	if len(post.Image) > 0 {
		util.Logger.Info().Msg("Processing Image Upload")
		mediaID := postImageToWordpress(post.Image, post.ImagePrompt)
		postData = map[string]interface{}{
			"title":          post.Title,
			"content":        post.Content,
			"status":         post.PublishStatus,
			"featured_media": mediaID,
			"excerpt":        post.Description,
		}
	} else {
		// Define the post data
		postData = map[string]interface{}{
			"title":   post.Title,
			"content": post.Content,
			"status":  post.PublishStatus,
			"excerpt": post.Description,
		}
	}
	err := doWordpressPost("/wp-json/wp/v2/posts", postData)
	if err != nil {
		util.Logger.Error().Err(err).Msg("Error creating post")
		return err
	}
	util.Logger.Info().Msg("Post created successfully!")
	return nil
}

func loadSettings() {
	newSettings, err := models.GetSettingsSimple()
	if err != nil {
		util.Logger.Error().Err(err).Msg("Error loading settings")
	} else {
		Settings = newSettings
	}
}

func loadTemplates() {
	newTemplates, err := models.GetTemplatesSimple()
	if err != nil {
		util.Logger.Error().Err(err).Msg("Error loading templates")
	} else {
		Templates = newTemplates
	}
}

func runSystemTests() {
	WordPressStatus = false
	OpenAiStatus = false
	SdStatus = false
	UnsplashStatus = false

	util.Logger.Info().Msg("Running system tests...")
	//Test WordPress Connection
	util.Logger.Info().Msg("Testing WordPress Connection...")
	_, err := getWpTitles()
	if err != nil {
		util.Logger.Error().Err(err).Msg("Error getting WordPress titles")
	} else {
		util.Logger.Info().Msg("WordPress Connection Successful!")
		WordPressStatus = true
	}
	//Test OpenAI Connection
	util.Logger.Info().Msg("Testing OpenAI Connection...")
	aiTestResp, err := openai.GenerateTestGreeting(Settings["OPENAI_API_KEY"], false, "You are running your start-up diagnostics, compose some humorous fake startup sequence events and a greeting as a sort of boot-up log and return them.  This response should be formatted an <ul> in HTML to be inserted into a status page.  Class \"font-monospace\" should be used on the text to give it a robotic feel.  The page already exists, we just need to drop in the HTML greeting inside the existing HTML page we have, so it should not include a body or head or close or open html tags, just the markup for the text itself within the page.", "You are Blog-o-Tron a sophisticated, AI-powered blogging robot.")
	if err != nil {
		util.Logger.Error().Err(err).Msg("Error testing OpenAI API")
	} else {
		util.Logger.Info().Msg("OpenAI Connection Successful!")
		OpenAiStatus = true
		Greeting = aiTestResp
	}
	//Test StableDiffusion Connection
	if Settings["IMG_MODE"] == "sd" {
		util.Logger.Info().Msg("Testing StableDiffusion Connection...")
		imgResp, err := generateSizedImage("An selfie image of Blog-o-Tron the blog-writing robot sitting in front of a computer in a futuristic lab waving at the camera.  Centered and in focus. Photo-realistic, Hyper-realistic, Portrait, Well Lit", 512, 512)
		if err != nil {
			util.Logger.Error().Err(err).Msg("Error testing StableDiffusion API")
		} else {
			util.Logger.Info().Msg("StableDiffusion Connection Successful!")
			SdStatus = true
			Selfie = imgResp
		}
	} else {
		util.Logger.Error().Msg("StableDiffusion is not enabled")
	}

	//Test Unsplash Connection
	util.Logger.Info().Msg("Testing Unsplash Connection...")
	_, err = unsplash.GetImageBySearch(Settings["UNSPLASH_ACCESS_KEY"], "robot")
	if err != nil {
		util.Logger.Error().Err(err).Msg("Error testing Unsplash API")
	} else {
		util.Logger.Info().Msg("Unsplash Connection Successful!")
		UnsplashStatus = true
	}
	LastTestTime = time.Now().Format("Jan 2, 2006 at 3:04pm (MST)")
	//Save Greeting to greeting.txt
	SaveStringToFile(Greeting, "greeting.txt")

	//Save Selfie to selfie.png
	SaveBytesAsPNG(Selfie, "selfie.png")

	//Get all status values as a CSV string along with timestamp as a human-readable string
	statusCsv := strconv.FormatBool(WordPressStatus) + "," + strconv.FormatBool(OpenAiStatus) + "," + strconv.FormatBool(SdStatus) + "," + strconv.FormatBool(UnsplashStatus) + "," + LastTestTime
	SaveStringToFile(statusCsv, "status.csv")
}

func loadCachedTestResults() {
	//Load Greeting from greeting.txt
	greeting, err := ReadStringFromFile("greeting.txt")
	if err != nil {
		util.Logger.Error().Err(err).Msg("Error loading greeting.txt")
	} else {
		Greeting = greeting
	}

	//Load Selfie from selfie.png
	selfie, err := ReadImageToBytes("selfie.png")
	if err != nil {
		util.Logger.Error().Err(err).Msg("Error loading selfie.png")
	} else {
		Selfie = selfie
	}

	//Load Status from status.csv
	statusCsv, err := ReadStringFromFile("status.csv")
	if err != nil {
		util.Logger.Error().Err(err).Msg("Error loading status.csv")
	} else {
		status := strings.Split(statusCsv, ",")
		WordPressStatus, _ = strconv.ParseBool(status[0])
		OpenAiStatus, _ = strconv.ParseBool(status[1])
		SdStatus, _ = strconv.ParseBool(status[2])
		UnsplashStatus, _ = strconv.ParseBool(status[3])
		LastTestTime = status[4]
	}
}

func SaveBytesAsPNG(bytes []byte, outputPath string) error {
	// Create a new file for writing the PNG
	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Write the byte array to the file
	_, err = file.Write(bytes)
	if err != nil {
		return err
	}

	util.Logger.Info().Msgf("PNG file saved successfully at: %s", outputPath)
	return nil
}

func ReadImageToBytes(filePath string) ([]byte, error) {
	// Read the image file
	imageData, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	return imageData, nil
}

func SaveStringToFile(content string, filePath string) error {
	// Write the string content to the file
	err := ioutil.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		return err
	}

	util.Logger.Info().Msgf("String saved successfully to: %s", filePath)
	return nil
}

func ReadStringFromFile(filePath string) (string, error) {
	// Read the file
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	return string(content), nil
}
