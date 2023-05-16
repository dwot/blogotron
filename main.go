package main

import (
	"bytes"
	"context"
	"embed"
	"encoding/base64"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/go-co-op/gocron"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/joho/godotenv"
	"github.com/meitarim/go-wordpress"
	"github.com/spf13/viper"
	"golang/api"
	"golang/config"
	"golang/models"
	"golang/openai"
	"golang/stablediffusion"
	"golang/unsplash"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"html/template"
)

//go:embed sql/migrations/*.sql
var MigrationSrc embed.FS

func main() {
	err := godotenv.Load()
	handleError(err)
	config.ParseConfig()

	//DB
	dbName := os.Getenv("BLOGOTRON_DB_NAME")
	err = models.ConnectDatabase(dbName)
	handleError(err)
	migSrc, err := iofs.New(MigrationSrc, "sql/migrations")
	handleError(err)
	err = models.MigrateDatabase(migSrc)
	handleError(err)

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
		cronSrv.Every("1h").Do(func() {
			//Check count of total open ideas
			ideaCount := models.GetOpenIdeaCount()
			fmt.Println("Idea Count: " + strconv.Itoa(ideaCount) + " Threshold: " + strconv.Itoa(iThreshold))
			if ideaCount < iThreshold {
				fullBrainstorm("10", false)
			}
		})
	}
	if autoPost == "true" {
		cronSrv.Every(autoPostInterval).Do(func() {
			//Get a Random Idea
			idea := models.GetRandomIdea()
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
			}

			post = writeArticle(post)
		})
	}

	//Thread Mgmt
	wg := new(sync.WaitGroup)
	wg.Add(3)
	fmt.Println("Starting Gin Server")
	go func() {
		apiGin.Run(":" + apiPort)
		wg.Done()
	}()
	fmt.Println("Started Gin Server")

	fmt.Println("Starting Web Server")
	go func() {
		err = http.ListenAndServe(":"+webPort, mux)
		handleError(err)
		wg.Done()
	}()
	fmt.Println("Started Web Server")

	fmt.Println("Starting Cron Server")
	go func() {
		//Start cron
		cronSrv.StartBlocking()
		wg.Done()
	}()
	fmt.Println("Started Cron Server")
	wg.Wait()

}

func generateImage(p string) []byte {
	var imgBytes []byte
	if p != "" {
		if os.Getenv("IMG_MODE") == "openai" {
			imgBytes = openai.GenerateImg(p)
		} else if os.Getenv("IMG_MODE") == "sd" {
			ctx := context.Background()
			images, err := stablediffusion.Generate(ctx, stablediffusion.SimpleImageRequest{
				Prompt:                            p,
				NegativePrompt:                    "watermark,border,blurry,duplicate",
				Styles:                            nil,
				Seed:                              -1,
				SamplerName:                       "DPM++ 2M",
				BatchSize:                         1,
				NIter:                             1,
				Steps:                             30,
				CfgScale:                          7,
				Width:                             512,
				Height:                            512,
				SNoise:                            0,
				OverrideSettings:                  struct{}{},
				OverrideSettingsRestoreAfterwards: false,
				SaveImages:                        true,
			})
			handleError(err)
			imgBytes = images.Images[0]
		} else {
			fmt.Println("image generation disabled")
		}
	}
	return imgBytes
}

func writeArticle(post Post) Post {
	newImgPrompt := ""
	article := ""
	title := ""
	if post.Prompt != "" {
		wpTmpl := template.Must(template.New("web-prompt").Parse(viper.GetString("config.prompt.web-prompt")))
		webPrompt := new(bytes.Buffer)
		err := wpTmpl.Execute(webPrompt, post)
		handleError(err)
		fmt.Println("Prompt is: ", webPrompt.String())
		articleResp, err := openai.GenerateArticle(post.UseGpt4, webPrompt.String(), viper.GetString("config.prompt.system-prompt"))
		handleError(err)
		article = articleResp
		//Attempt to parse out title from h1 tag
		if strings.Contains(article, "<h1>") && strings.Contains(article, "</h1>") && strings.HasPrefix(article, "<h1>") {
			fmt.Println("Detected Title")
			tempTitle := strings.Split(strings.Split(article, "<h1>")[1], "</h1>")[0]
			if tempTitle == "Introduction" {
				fmt.Println("Title is Introduction, skipping")
			} else {
				title = tempTitle
				//Remove title from article
				article = strings.Replace(article, "<h1>"+title+"</h1>", "", 1)
				fmt.Println("Removed Title: " + title)
				//Remove any leading newlines from article
				article = strings.TrimPrefix(article, "\n")
			}
		}
		if title == "" {
			if !post.ConceptAsTitle {
				titleResp, err := openai.GenerateTitle(false, article, viper.GetString("config.prompt.title-prompt"), viper.GetString("config.prompt.system-prompt"))
				handleError(err)
				title = titleResp
			} else {
				title = post.Prompt
			}
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
			imgGenResp, err := openai.GenerateTitle(false, title, viper.GetString("config.prompt.imggen-prompt"), viper.GetString("config.prompt.system-prompt"))
			handleError(err)
			fmt.Println("Img Gen is: ", imgGenResp)
			imgGenResp = strings.Replace(imgGenResp, "\"", "", 1)
			imgGenResp = strings.Replace(imgGenResp, "Create an image of ", "", 1)
			post.ImagePrompt = imgGenResp
		}
		fmt.Println("Img Prompt in is: ", post.ImagePrompt)
		imgTmpl := template.Must(template.New("img-prompt").Parse(viper.GetString("config.prompt.img-prompt")))
		imgBuiltPrompt := new(bytes.Buffer)
		err := imgTmpl.Execute(imgBuiltPrompt, post)
		handleError(err)
		newImgPrompt = imgBuiltPrompt.String()
		fmt.Println("Img Prompt out is: ", newImgPrompt)
		post.Image = generateImage(newImgPrompt)
	} else if post.Error == "" && post.DownloadImg && post.ImgUrl != "" {
		response, err := http.Get(post.ImgUrl)
		handleError(err)
		defer func() {
			response.Body.Close()
		}()
		if response.StatusCode != 200 {
			post.Error = "Bad response code downloading image: " + strconv.Itoa(response.StatusCode)
		}
		imgBytes, err := io.ReadAll(response.Body)
		handleError(err)
		post.Image = imgBytes
	} else if post.Error == "" && post.UnsplashImg && post.UnsplashSearch != "" {
		imgBytes := unsplash.GetImageBySearch(post.UnsplashSearch)
		post.Image = imgBytes
	} else if post.Error == "" && post.UnsplashImg && post.UnsplashSearch == "" {
		imgSearchResp, err := openai.GenerateTitle(false, title, viper.GetString("config.prompt.imgsearch-prompt"), viper.GetString("config.prompt.system-prompt"))
		handleError(err)
		fmt.Println("Img Search is: ", imgSearchResp)
		post.UnsplashSearch = imgSearchResp
		imgBytes := unsplash.GetImageBySearch(imgSearchResp)
		post.Image = imgBytes
	}
	post.ImageB64 = base64.StdEncoding.EncodeToString(post.Image)
	postToWordpress(post)
	models.SetIdeaWritten(post.IdeaId)
	return post
}

func postToWordpress(post Post) *wordpress.Post {
	client := wordpress.NewClient(&wordpress.Options{
		BaseAPIURL: os.Getenv("WP_URL") + "/wp-json/wp/v2",
		Username:   os.Getenv("WP_USERNAME"),
		Password:   os.Getenv("WP_PASSWORD"),
	})
	newPost := &wordpress.Post{
		Title: wordpress.Title{
			Raw: post.Title,
		},
		Content: wordpress.Content{
			Raw: post.Content,
		},
		Status: post.PublishStatus,
	}
	if len(post.Image) > 0 {
		fmt.Println("Processing Image Upload")

		media := &wordpress.MediaUploadOptions{
			Filename:    "test-media.png",
			ContentType: "image/png",
			Data:        post.Image,
		}
		newMedia, resp, _, err := client.Media().Create(media)
		handleError(err)
		if resp.StatusCode != http.StatusCreated {
			fmt.Println("Expected 201 Created, got" + resp.Status)
		}
		if newMedia != nil {
			newPost.FeaturedImage = newMedia.ID
		}
	}

	newPost, res, _, err := client.Posts().Create(newPost)
	handleError(err)
	fmt.Println(res)
	//fmt.Printf("%+v\n", post)
	return newPost
}

func generateIdeas(ideaCount string, builtConcept string, useGpt4 bool, sid int, ideaConcept string) {
	prompt := Prompt{
		IdeaCount:   ideaCount,
		IdeaConcept: builtConcept,
	}
	ideaTmpl := template.Must(template.New("idea-prompt").Parse(viper.GetString("config.prompt.idea-prompt")))
	ideaPrompt := new(bytes.Buffer)
	err := ideaTmpl.Execute(ideaPrompt, prompt)
	handleError(err)
	fmt.Println("Prompt is: ", ideaPrompt.String())
	ideaResp, err := openai.GenerateArticle(useGpt4, ideaPrompt.String(), viper.GetString("config.prompt.system-prompt"))
	handleError(err)
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
	handleError(err)
	fmt.Println("Topic Prompt is: ", ideaPrompt.String())
	ideaResp, err := openai.GenerateArticle(useGpt4, ideaPrompt.String(), viper.GetString("config.prompt.system-prompt"))
	handleError(err)
	ideaResp = strings.ReplaceAll(ideaResp, "\n", "")
	fmt.Println("Idea Brainstorm Results: " + ideaResp)
	ideaList := strings.Split(ideaResp, "|")
	for _, value := range ideaList {
		builtConcept := "The topic for the ideas is: \"" + value + "\"."
		generateIdeas(ideaCount, builtConcept, useGpt4, 0, value)
	}
}

func handleError(err error) {
	if err != nil {
		fmt.Println("Error: ", err.Error())
	}
}
