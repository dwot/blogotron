package main

import (
	"bytes"
	"context"
	"fmt"
	"github.com/joho/godotenv"
	wordpress "github.com/meitarim/go-wordpress"
	"github.com/spf13/viper"
	"golang/config"
	"golang/openai"
	"golang/stablediffusion"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"os"
)

type Post struct {
	Title       string `json:"title"`
	Content     string `json:"content"`
	Description string `json:"description"`
	Image       []byte `json:"image"`
	Prompt      string `json:"prompt"`
	ImagePrompt string `json:"imageprompt"`
	Error       string `json:"error"`
}

var tpl = template.Must(template.ParseFiles("templates\\index.html"))

func writeArticle(p string, p2 string) Post {
	article, err := openai.GenerateArticle(p, viper.GetString("config.prompt.system-prompt"))
	if err != nil {
		panic(err)
	}

	title, err := openai.GenerateTitle(article, viper.GetString("config.prompt.title-prompt"), viper.GetString("config.prompt.system-prompt"))
	if err != nil {
		panic(err)
	}
	fmt.Println(title)

	var imgBytes []byte
	if p2 != "" {
		fmt.Println("Negatives:" + viper.GetString("config.settings.image-negatives"))
		fmt.Println("OK We're going to get an image!" + p2)
		if os.Getenv("IMG_MODE") == "openai" {
			imgBytes = openai.GenerateImg(p2)
		} else if os.Getenv("IMG_MODE") == "sd" {
			ctx := context.Background()
			imgs, err := stablediffusion.Generate(ctx, stablediffusion.SimpleImageRequest{
				Prompt:                            p2,
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
			imgBytes = imgs.Images[0]
		} else {
			fmt.Println("image generation disabled")
		}
	}

	post := Post{
		Title:       title,
		Content:     article,
		Image:       imgBytes,
		Prompt:      p,
		ImagePrompt: p2,
	}
	return post
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	config.ParseConfig()

	port := os.Getenv("BLOGOTRON_PORT")
	if port == "" {
		port = "8666"
	}
	fs := http.FileServer(http.Dir("assets"))
	mux := http.NewServeMux()

	mux.HandleFunc("/", indexHandler)
	mux.HandleFunc("/write", writeHandler)
	mux.HandleFunc("/imgtest", imageHandler)
	mux.HandleFunc("/test", testHandler)
	mux.Handle("/assets/", http.StripPrefix("/assets/", fs))
	http.ListenAndServe(":"+port, mux)
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	buf := &bytes.Buffer{}
	err := tpl.Execute(buf, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	buf.WriteTo(w)
}

func testHandler(w http.ResponseWriter, r *http.Request) {
	var newPost Post
	newPost.Prompt = "testaroni"
	wpTmpl := template.Must(template.New("web-prompt").Parse(viper.GetString("config.prompt.web-prompt")))
	webPrompt := new(bytes.Buffer)
	wpErr := wpTmpl.Execute(webPrompt, newPost)
	if wpErr != nil {
		panic(wpErr)
	}
	fmt.Println("Test that shit" + newPost.Prompt)
	fmt.Println("TEST:" + webPrompt.String())
	tpl.Execute(w, nil)
}

func imageHandler(w http.ResponseWriter, r *http.Request) {
	var imgBytes []byte
	p2 := "a portrait of a blue jay"
	fmt.Println("OK We're going to get an image!")
	if os.Getenv("IMG_MODE") == "openai" {
		imgBytes = openai.GenerateImg(p2)
	} else if os.Getenv("IMG_MODE") == "sd" {
		ctx := context.Background()
		imgs, err := stablediffusion.Generate(ctx, stablediffusion.SimpleImageRequest{
			Prompt:                            p2,
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
		imgBytes = imgs.Images[0]
	}
	fmt.Println(len(imgBytes))
	tpl.Execute(w, nil)
}

func writeHandler(w http.ResponseWriter, r *http.Request) {
	u, err := url.Parse(r.URL.String())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	promptEntry := ""
	imgPrompt := ""
	switch r.Method {
	case "GET":
		params := u.Query()
		promptEntry = params.Get("q")
		imgPrompt = params.Get("m")

	case "POST":
		promptEntry = r.FormValue("q")
		imgPrompt = r.FormValue("m")
	}

	var newPost Post
	newImgPrompt := ""
	if promptEntry != "" {
		newPost.Prompt = promptEntry
		wpTmpl := template.Must(template.New("web-prompt").Parse(viper.GetString("config.prompt.web-prompt")))
		webPrompt := new(bytes.Buffer)
		wpErr := wpTmpl.Execute(webPrompt, newPost)
		if wpErr != nil {
			panic(wpErr)
		}
		if imgPrompt != "" {
			newPost.ImagePrompt = imgPrompt
			fmt.Println("Img Prompt in is: ", imgPrompt)
			imgTmpl := template.Must(template.New("img-prompt").Parse(viper.GetString("config.prompt.img-prompt")))
			imgBuiltPrompt := new(bytes.Buffer)
			imgErr := imgTmpl.Execute(imgBuiltPrompt, newPost)
			if imgErr != nil {
				panic(imgErr)
			}
			newImgPrompt = imgBuiltPrompt.String()
			fmt.Println("Img Prompt out is: ", newImgPrompt)
		}

		fmt.Println("Prompt is: ", webPrompt.String())
		newPost = writeArticle(webPrompt.String(), newImgPrompt)
		postToWordpress(newPost)
	} else {
		newPost.Error = "Please input an article idea first."
	}

	buf := &bytes.Buffer{}
	err = tpl.Execute(buf, newPost)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	buf.WriteTo(w)
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
		newMedia, resp, body, err := client.Media().Create(media)
		if err != nil {
			fmt.Println("Should not return error:" + err.Error())
		}
		if resp.StatusCode != http.StatusCreated {
			fmt.Println("Expected 201 Created, got" + resp.Status)
		}
		if body == nil {
			fmt.Println("Should not return nil body")
		}
		if newMedia == nil {
			fmt.Println("Should not return nil newMedia")
		}
		fmt.Println(resp)
		newPost.FeaturedImage = newMedia.ID
	}

	newPost, res, _, err := client.Posts().Create(newPost)
	if err != nil {
		panic(err)
	}
	fmt.Println(res)
	//fmt.Printf("%+v\n", post)
	return newPost
}
