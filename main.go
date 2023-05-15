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
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"html/template"
)

//go:embed sql/migrations/*.sql
var MigrationSrc embed.FS

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	config.ParseConfig()

	//DB
	dbName := os.Getenv("BLOGOTRON_DB_NAME")
	err = models.ConnectDatabase(dbName)
	if err != nil {
		log.Fatal("Error connecting DB " + err.Error())
	}
	migSrc, err := iofs.New(MigrationSrc, "sql/migrations")
	if err != nil {
		log.Fatal("Error loading db migration " + err.Error())
	}
	err = models.MigrateDatabase(migSrc)
	if err != nil {
		log.Fatal("Error migrating DB " + err.Error())
	}

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
	cronSrv := gocron.NewScheduler(time.UTC)
	if autoPost == "true" {
		cronSrv.Every(autoPostInterval).Do(func() {
			fmt.Println("Posting Time")
			//Get a Random Idea
			idea := models.GetRandomIdea()
			//Create a new post from the idea
			iLen := 1000
			publishStatus := "publish"
			unsplashSearch := ""

			post := Post{
				Title:          "",
				Content:        "",
				Description:    "",
				Image:          []byte{},
				Prompt:         idea.IdeaText,
				ImagePrompt:    "",
				Error:          "",
				ImageB64:       "",
				Length:         iLen,
				PublishStatus:  publishStatus,
				UseGpt4:        false,
				ConceptAsTitle: false,
				IncludeYt:      false,
				YtUrl:          "",
				GenerateImg:    false,
				DownloadImg:    false,
				ImgUrl:         "",
				UnsplashImg:    true,
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
		if err != nil {
			fmt.Println("Error starting http server:" + err.Error())
		}
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
		fmt.Println("Negatives:" + viper.GetString("config.settings.image-negatives"))
		fmt.Println("OK We're going to get an image!" + p)
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
			if err != nil {
				fmt.Println("Err" + err.Error())
			}
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
		wpErr := wpTmpl.Execute(webPrompt, post)
		if wpErr != nil {
			post.Error = "Error generating web prompt from template: " + wpErr.Error()
		}
		fmt.Println("Prompt is: ", webPrompt.String())
		articleResp, err := openai.GenerateArticle(post.UseGpt4, webPrompt.String(), viper.GetString("config.prompt.system-prompt"))
		if err != nil {
			post.Error = "Error generating article from OpenAI API: " + err.Error()
		}
		article = articleResp
		if post.ConceptAsTitle {
			titleResp, err := openai.GenerateTitle(false, article, viper.GetString("config.prompt.title-prompt"), viper.GetString("config.prompt.system-prompt"))
			if err != nil {
				post.Error = "Error generating title from OpenAI API: " + err.Error()
			}
			title = titleResp
		} else {
			title = post.Prompt
		}
		if post.IncludeYt && post.YtUrl != "" {
			article = article + "\n<p>[embed]" + post.YtUrl + "[/embed]</p>"
		}
		post.Content = article
		post.Title = title
	} else {
		post.Error = "Please input an article idea first."
	}

	if post.Error == "" && post.GenerateImg && post.ImagePrompt != "" {
		fmt.Println("Img Prompt in is: ", post.ImagePrompt)
		imgTmpl := template.Must(template.New("img-prompt").Parse(viper.GetString("config.prompt.img-prompt")))
		imgBuiltPrompt := new(bytes.Buffer)
		imgErr := imgTmpl.Execute(imgBuiltPrompt, post)
		if imgErr != nil {
			post.Error = "Error generating image: " + imgErr.Error()
		}
		newImgPrompt = imgBuiltPrompt.String()
		fmt.Println("Img Prompt out is: ", newImgPrompt)
		post.Image = generateImage(newImgPrompt)
	} else if post.Error == "" && post.DownloadImg && post.ImgUrl != "" {
		response, err := http.Get(post.ImgUrl)
		if err != nil {
			post.Error = "Error downloading image: " + err.Error()
		}
		defer func() {
			response.Body.Close()
		}()
		if response.StatusCode != 200 {
			post.Error = "Bad response code downloading image: " + strconv.Itoa(response.StatusCode)
		}
		imgBytes, err := io.ReadAll(response.Body)
		if err != nil {
			post.Error = "Error reading image bytes: " + err.Error()
		}
		post.Image = imgBytes
	} else if post.Error == "" && post.UnsplashImg && post.UnsplashSearch != "" {
		imgBytes := unsplash.GetImageBySearch(post.UnsplashSearch)
		post.Image = imgBytes
	} else if post.Error == "" && post.UnsplashImg && post.UnsplashSearch == "" {
		imgSearchResp, err := openai.GenerateTitle(false, title, viper.GetString("config.prompt.imgsearch-prompt"), viper.GetString("config.prompt.system-prompt"))
		if err != nil {
			post.Error = "Error generating imgSearch from OpenAI API: " + err.Error()
		}
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
		if err != nil {
			fmt.Println("Should not return error:" + err.Error())
		}
		if resp.StatusCode != http.StatusCreated {
			fmt.Println("Expected 201 Created, got" + resp.Status)
		}
		if newMedia != nil {
			newPost.FeaturedImage = newMedia.ID
		}
	}

	newPost, res, _, err := client.Posts().Create(newPost)
	if err != nil {
		fmt.Println("Error posting to WordPress:" + err.Error())
	}
	fmt.Println(res)
	//fmt.Printf("%+v\n", post)
	return newPost
}
