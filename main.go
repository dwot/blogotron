package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"github.com/joho/godotenv"
	"github.com/meitarim/go-wordpress"
	"github.com/spf13/viper"
	"golang/config"
	"golang/openai"
	"golang/stablediffusion"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
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
}

var resultsTpl = template.Must(template.ParseFiles("templates\\results.html"))
var menuTpl = template.Must(template.ParseFiles("templates\\menu.html"))

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

	mux.HandleFunc("/", menuHandler)
	mux.HandleFunc("/write", writeHandler)
	mux.HandleFunc("/menu", menuHandler)
	mux.Handle("/assets/", http.StripPrefix("/assets/", fs))
	err = http.ListenAndServe(":"+port, mux)
	if err != nil {
		fmt.Println("Error starting http server:" + err.Error())
	}
}

func menuHandler(w http.ResponseWriter, _ *http.Request) {
	buf := &bytes.Buffer{}
	err := menuTpl.Execute(buf, nil)
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
	promptEntry := r.FormValue("articleConcept")
	imgPrompt := r.FormValue("imagePrompt")
	imgUrl := r.FormValue("imageUrl")
	downloadImg := r.FormValue("downloadImage")
	generateImg := r.FormValue("generateImage")
	includeYt := r.FormValue("includeYt")
	ytUrl := r.FormValue("ytUrl")
	article := ""
	title := ""
	var imgBytes []byte

	post := Post{
		Image:       imgBytes,
		Prompt:      promptEntry,
		ImagePrompt: imgPrompt,
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
		articleResp, err := openai.GenerateArticle(webPrompt.String(), viper.GetString("config.prompt.system-prompt"))
		if err != nil {
			post.Error = "Error generating article from OpenAI API: " + err.Error()
		}
		article = articleResp
		titleResp, err := openai.GenerateTitle(article, viper.GetString("config.prompt.title-prompt"), viper.GetString("config.prompt.system-prompt"))
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
		defer response.Body.Close()
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

	_, wtErr := buf.WriteTo(w)
	if wtErr != nil {
		fmt.Println("Error rendering results html:" + wtErr.Error())
	}
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
