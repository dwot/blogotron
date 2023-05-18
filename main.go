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
	"github.com/joho/godotenv"
	"github.com/spf13/viper"
	"golang/api"
	"golang/config"
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

func main() {
	util.Init()
	util.Logger.Info().Msg("Starting Blogotron")
	err := godotenv.Load()
	util.HandleErrorAndTerminate(err, "Could not load .env file")
	err = config.ParseConfig()
	util.HandleErrorAndTerminate(err, "Could not load config.yml file")

	//DB
	dbName := os.Getenv("BLOGOTRON_DB_NAME")
	err = models.ConnectDatabase(dbName)
	util.HandleErrorAndTerminate(err, "Could not connect to database file "+dbName)
	migSrc, err := iofs.New(MigrationSrc, "sql/migrations")
	util.HandleErrorAndTerminate(err, "Could not load migrations")
	err = models.MigrateDatabase(migSrc)
	util.HandleErrorAndTerminate(err, "Could not migrate database")

	//API
	apiPort := os.Getenv("BLOGOTRON_API_PORT")
	if apiPort == "" {
		apiPort = "8667"
	}
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
	webPort := os.Getenv("BLOGOTRON_PORT")
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
	mux.HandleFunc("/test", testHandler)
	mux.Handle("/assets/", http.StripPrefix("/assets/", fs))

	//Cron Service
	autoPost := os.Getenv("AUTO_POST_ENABLE")
	autoPostInterval := os.Getenv("AUTO_POST_INTERVAL")
	autoPostImgEngine := os.Getenv("AUTO_POST_IMG_ENGINE")
	autoPostLen := os.Getenv("AUTO_POST_LEN")
	autoPostState := os.Getenv("AUTO_POST_STATE")
	lowIdeaThreshold := os.Getenv("LOW_IDEA_THRESHOLD")
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
				util.HandleError(err, "Could not get random idea")
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
				util.HandleError(err, "Could not write article")
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
		util.HandleError(err, "Error starting web server")
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

func generateImage(p string) []byte {
	var imgBytes []byte
	imgWidth := os.Getenv("IMG_WIDTH")
	imgHeight := os.Getenv("IMG_HEIGHT")
	imgSampler := os.Getenv("IMG_SAMPLER")
	imgNegativePrompts := os.Getenv("IMG_NEGATIVE_PROMPTS")
	imgSteps := os.Getenv("IMG_STEPS")

	iWidth, err := strconv.Atoi(imgWidth)
	if err != nil {
		iWidth = 512
	}
	iHeight, err := strconv.Atoi(imgHeight)
	if err != nil {
		iHeight = 512
	}
	iSteps, err := strconv.Atoi(imgSteps)
	if err != nil {
		iSteps = 30
	}

	if p != "" {
		if os.Getenv("IMG_MODE") == "openai" {
			imgBytes = openai.GenerateImg(p)
		} else if os.Getenv("IMG_MODE") == "sd" {
			ctx := context.Background()
			images, err := stablediffusion.Generate(ctx, stablediffusion.SimpleImageRequest{
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
			})
			util.HandleError(err, "Error generating image")
			imgBytes = images.Images[0]
		}
	}
	return imgBytes
}

func writeArticle(post Post) (error, Post) {
	newImgPrompt := ""
	article := ""
	title := ""
	if post.Prompt != "" {
		if post.Keyword == "" {
			kwTmpl := template.Must(template.New("keyword-prompt").Parse(viper.GetString("config.prompt.keyword-prompt")))
			keywordPrompt := new(bytes.Buffer)
			err := kwTmpl.Execute(keywordPrompt, post)
			if err != nil {
				return err, post
			}
			keywordResp, err := openai.GenerateArticle(post.UseGpt4, keywordPrompt.String(), viper.GetString("config.prompt.system-prompt"))
			if err != nil {
				return err, post
			}
			post.Keyword = keywordResp
		}
		wpTmpl := template.Must(template.New("web-prompt").Parse(viper.GetString("config.prompt.web-prompt")))
		webPrompt := new(bytes.Buffer)
		err := wpTmpl.Execute(webPrompt, post)
		if err != nil {
			return err, post
		}
		util.Logger.Info().Msg("Generating Article from Prompt" + webPrompt.String() + "")
		articleResp, err := openai.GenerateArticle(post.UseGpt4, webPrompt.String(), viper.GetString("config.prompt.system-prompt"))
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
				titleResp, err := openai.GenerateTitle(false, article, viper.GetString("config.prompt.title-prompt"), viper.GetString("config.prompt.system-prompt"))
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
			descTmpl := template.Must(template.New("description-prompt").Parse(viper.GetString("config.prompt.description-prompt")))
			descPrompt := new(bytes.Buffer)
			err := descTmpl.Execute(descPrompt, post)
			descResp, err := openai.GenerateTitle(false, article, descPrompt.String(), viper.GetString("config.prompt.system-prompt"))
			if err != nil {
				return err, post
			}
			post.Description = descResp
		}
		if post.IncludeYt && post.YtUrl != "" {
			article = article + "\n<p>[embed]" + post.YtUrl + "[/embed]</p>"
		}
		post.Content = article
		post.Title = title
	} else {
		post.Error = "Please input an article idea first."
	}

	if post.Error == "" && post.GenerateImg {
		if post.ImagePrompt == "" {
			igTmpl := template.Must(template.New("imggen-prompt").Parse(viper.GetString("config.prompt.imggen-prompt")))
			imgGenPrompt := new(bytes.Buffer)
			err := igTmpl.Execute(imgGenPrompt, post)
			imgGenResp, err := openai.GenerateTitle(false, title, imgGenPrompt.String(), viper.GetString("config.prompt.system-prompt"))
			if err != nil {
				return err, post
			}
			imgGenResp = strings.Replace(imgGenResp, "\"", "", 1)
			imgGenResp = strings.Replace(imgGenResp, "Create an image of ", "", 1)
			imgGenResp = strings.Replace(imgGenResp, "Can you create an image of ", "", 1)
			post.ImagePrompt = imgGenResp
		}
		util.Logger.Info().Msg("Img Prompt in is: " + post.ImagePrompt)
		imgTmpl := template.Must(template.New("img-prompt").Parse(viper.GetString("config.prompt.img-prompt")))
		imgBuiltPrompt := new(bytes.Buffer)
		err := imgTmpl.Execute(imgBuiltPrompt, post)
		if err != nil {
			return err, post
		}
		newImgPrompt = imgBuiltPrompt.String()
		util.Logger.Info().Msg("Img Prompt Out is: " + newImgPrompt)
		post.Image = generateImage(newImgPrompt)
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
		imgBytes := unsplash.GetImageBySearch(post.UnsplashSearch)
		post.Image = imgBytes
	} else if post.Error == "" && post.UnsplashImg && post.UnsplashSearch == "" {
		imgSearchResp, err := openai.GenerateTitle(false, title, viper.GetString("config.prompt.imgsearch-prompt"), viper.GetString("config.prompt.system-prompt"))
		if err != nil {
			return err, post
		}
		post.UnsplashSearch = imgSearchResp
		imgBytes := unsplash.GetImageBySearch(imgSearchResp)
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
	ideaTmpl := template.Must(template.New("idea-prompt").Parse(viper.GetString("config.prompt.idea-prompt")))
	ideaPrompt := new(bytes.Buffer)
	err := ideaTmpl.Execute(ideaPrompt, prompt)
	util.HandleError(err, "Error executing idea template")
	util.Logger.Info().Msg("Prompt is: " + ideaPrompt.String())
	ideaResp, err := openai.GenerateArticle(useGpt4, ideaPrompt.String(), viper.GetString("config.prompt.system-prompt"))
	util.HandleError(err, "Error generating ideas")
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
			util.HandleError(err, "Error adding idea")
		}
	}
}

func fullBrainstorm(ideaCount string, useGpt4 bool) {
	conceptList := ""
	builtTopic := ""
	concepts, _ := models.GetIdeaConcepts()
	series, _ := models.GetSeries()
	for _, concept := range concepts {
		conceptList = conceptList + ", " + concept
	}
	for _, s := range series {
		conceptList = conceptList + ", " + s.SeriesPrompt
	}
	builtTopic = builtTopic + " Previous topics used include: " + conceptList + "."
	ideaTmpl := template.Must(template.New("topic-prompt").Parse(viper.GetString("config.prompt.topic-prompt")))
	ideaPrompt := new(bytes.Buffer)
	prompt := Prompt{
		IdeaCount:   ideaCount,
		IdeaConcept: builtTopic,
	}
	err := ideaTmpl.Execute(ideaPrompt, prompt)
	util.HandleError(err, "Error executing idea template")
	util.Logger.Info().Msg("Topic Prompt is: " + ideaPrompt.String())
	ideaResp, err := openai.GenerateArticle(useGpt4, ideaPrompt.String(), viper.GetString("config.prompt.system-prompt"))
	util.HandleError(err, "Error generating ideas")
	ideaResp = strings.ReplaceAll(ideaResp, "\n", "")
	util.Logger.Info().Msg("Idea Brainstorm Results: " + ideaResp)
	ideaList := strings.Split(ideaResp, "|")
	for _, value := range ideaList {
		builtConcept := "The topic for the ideas is: \"" + value + "\"."
		generateIdeas(ideaCount, builtConcept, useGpt4, 0, value)
	}
}

func getWpTitles() ([]string, error) {
	// Create an HTTP client
	client := &http.Client{}

	// Define the URL and request method
	url := os.Getenv("WP_URL") + "/wp-json/wp/v2/posts?per_page=100"
	method := "GET"

	// Define the authentication credentials
	username := os.Getenv("WP_USERNAME")
	password := os.Getenv("WP_PASSWORD")

	// Create a request
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		util.HandleError(err, "Error creating request for title list")
		return nil, err
	}

	// Set the authentication header
	req.SetBasicAuth(username, password)

	// Send the request
	res, err := client.Do(req)
	if err != nil {
		util.HandleError(err, "Error sending request for title list")
		return nil, err
	}

	// Read the response body
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		util.HandleError(err, "Error reading response body for title list")
		return nil, err
	}

	// Close the response body
	res.Body.Close()

	// Parse the JSON data
	var posts []map[string]interface{}
	err = json.Unmarshal(body, &posts)
	if err != nil {
		util.HandleError(err, "Error parsing JSON for title list")
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
	url := os.Getenv("WP_URL") + endPoint //"/wp-json/wp/v2/posts"
	method := "POST"

	// Define the authentication credentials
	username := os.Getenv("WP_USERNAME")
	password := os.Getenv("WP_PASSWORD")

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
	req.Header.Set("Host", strings.ReplaceAll(strings.ReplaceAll(os.Getenv("WP_URL"), "https://", ""), "http://", ""))
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
	util.HandleError(err, "Error writing alt text field")

	// Close the multipart writer
	err = writer.Close()
	if err != nil {
		util.Logger.Error().Err(err).Msg("Error closing multipart writer")
		return 0
	}
	// Create an HTTP client
	client := &http.Client{}

	// Define the URL and request method
	url := os.Getenv("WP_URL") + "/wp-json/wp/v2/media"
	method := "POST"

	// Define the authentication credentials
	username := os.Getenv("WP_USERNAME")
	password := os.Getenv("WP_PASSWORD")

	// Create a request with the multipart body
	req, err := http.NewRequest(method, url, body)
	util.HandleError(err, "Error creating request for image upload")

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
	req.Header.Set("Host", strings.ReplaceAll(strings.ReplaceAll(os.Getenv("WP_URL"), "https://", ""), "http://", ""))
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

	mediaID := mediaResp.ID
	util.Logger.Info().Msg("Image uploaded successfully! Media ID:" + strconv.Itoa(mediaID))
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
