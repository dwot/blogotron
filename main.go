package main

import (
	"bytes"
	"fmt"
	"github.com/joho/godotenv"
	wordpress "github.com/meitarim/go-wordpress"
	"github.com/spf13/viper"
	"golang/config"
	"golang/openai"
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
		fmt.Println("OK We're going to get an image!")
		imgBytes = openai.GenerateImg(p2)
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
	if promptEntry != "" {
		builtPrompt := "Write an article about \"" + promptEntry + "\". Use Wordpress html to format your article.  For SEO purposes, please use headings and bold text."

		fmt.Println("Prompt is: ", builtPrompt)
		newPost = writeArticle(builtPrompt, imgPrompt)
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
