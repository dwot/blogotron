package main

import (
	"context"
	"embed"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/joho/godotenv"
	"github.com/meitarim/go-wordpress"
	"github.com/spf13/viper"
	"golang/api"
	"golang/config"
	"golang/models"
	"golang/openai"
	"golang/stablediffusion"
	"log"
	"net/http"
	"os"
	"sync"
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
	mux.HandleFunc("/plan", planHandler)
	mux.HandleFunc("/aiIdea", aiIdeaHandler)
	mux.HandleFunc("/idea", ideaHandler)
	mux.HandleFunc("/ideaSave", ideaSaveHandler)
	mux.HandleFunc("/ideaDel", ideaRemoveHandler)
	mux.Handle("/assets/", http.StripPrefix("/assets/", fs))

	//Thread Mgmt
	wg := new(sync.WaitGroup)
	wg.Add(2)
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

func postToWordpress(post Post) *wordpress.Post {
	client := wordpress.NewClient(&wordpress.Options{
		BaseAPIURL: os.Getenv("WP_URL") + "/wp-json/wp/v2",
		Username:   os.Getenv("WP_USERNAME"),
		Password:   os.Getenv("WP_PASSWORD"),
	})
	newPost := &wordpress.Post{Title: wordpress.Title{
		Raw: post.Title,
	},
		Content: wordpress.Content{
			Raw: post.Content,
		},
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
