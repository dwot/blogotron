package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/meitarim/go-wordpress"
	"github.com/spf13/viper"
	"golang/api"
	"golang/config"
	"golang/models"
	"golang/openai"
	"golang/stablediffusion"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
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
	Length      int    `json:"article-length"`
}

var resultsTpl = template.Must(template.ParseFiles("templates\\results.html"))
var writeTpl = template.Must(template.ParseFiles("templates\\write.html"))
var planTpl = template.Must(template.ParseFiles("templates\\plan.html"))
var indexTpl = template.Must(template.ParseFiles("templates\\index.html"))
var ideaTpl = template.Must(template.ParseFiles("templates\\idea.html"))

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

type PageData struct {
	ErrorCode   string `json:"error-code"`
	GPT4Enabled bool   `json:"gpt4-enabled"`
	IdeaText    string `json:"idea-text"`
	IdeaId      string `json:"idea-id"`
}

func indexHandler(w http.ResponseWriter, _ *http.Request) {
	indexData := PageData{
		ErrorCode: "",
	}
	buf := &bytes.Buffer{}
	err := indexTpl.Execute(buf, indexData)
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
	ideaId := r.FormValue("ideaId")
	id, err := strconv.Atoi(ideaId)
	if err != nil {
		id = 0
	}
	ideaText := ""
	if id > 0 {
		idea, dbErr := models.GetIdeaById(ideaId)
		if dbErr != nil {
			fmt.Println("Err looking up idea by id:" + dbErr.Error())
		}
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
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_, err = buf.WriteTo(w)
	if err != nil {
		fmt.Println("Err rendering menu:" + err.Error())
	}
}

type PlanData struct {
	ErrorCode string        `json:"error-code"`
	Ideas     []models.Idea `json:"ideas"`
}

func planHandler(w http.ResponseWriter, _ *http.Request) {
	ideas, err := models.GetOpenIdeas()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	planData := PlanData{
		ErrorCode: "",
		Ideas:     ideas,
	}
	buf := &bytes.Buffer{}
	err = planTpl.Execute(buf, planData)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_, err = buf.WriteTo(w)
	if err != nil {
		fmt.Println("Err rendering menu:" + err.Error())
	}
}

func ideaHandler(w http.ResponseWriter, _ *http.Request) {
	ideas, err := models.GetOpenIdeas()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	planData := PlanData{
		ErrorCode: "",
		Ideas:     ideas,
	}
	buf := &bytes.Buffer{}
	err = ideaTpl.Execute(buf, planData)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_, err = buf.WriteTo(w)
	if err != nil {
		fmt.Println("Err rendering menu:" + err.Error())
	}
}

func aiIdeaHandler(w http.ResponseWriter, _ *http.Request) {
	useGpt4 := false
	ideaResp, err := openai.GenerateArticle(useGpt4, viper.GetString("config.prompt.idea-prompt"), viper.GetString("config.prompt.system-prompt"))
	ideaResp = strings.ReplaceAll(ideaResp, "\n", "")
	fmt.Println("Idea Brainstorm Results: " + ideaResp)
	ideaList := strings.Split(ideaResp, "|")
	for index, value := range ideaList {
		fmt.Printf("Index: %d, Value: %s\n", index, value)
		if strings.TrimSpace(value) != "" {
			models.AddIdeaText(strings.TrimSpace(value))
		}
	}
	ideas, err := models.GetOpenIdeas()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	planData := PlanData{
		ErrorCode: "",
		Ideas:     ideas,
	}
	buf := &bytes.Buffer{}
	err = planTpl.Execute(buf, planData)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_, err = buf.WriteTo(w)
	if err != nil {
		fmt.Println("Err rendering menu:" + err.Error())
	}
}

func ideaSaveHandler(w http.ResponseWriter, r *http.Request) {
	ideaText := r.FormValue("ideaText")
	ideaId := r.FormValue("ideaId")
	id, convErr := strconv.Atoi(ideaId)
	if convErr != nil {
		id = 0
	}
	if id > 0 {
		//Update by Id
	} else {
		//Insert New
		models.AddIdeaText(ideaText)
	}

	ideas, err := models.GetOpenIdeas()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	planData := PlanData{
		ErrorCode: "",
		Ideas:     ideas,
	}
	buf := &bytes.Buffer{}
	err = planTpl.Execute(buf, planData)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_, err = buf.WriteTo(w)
	if err != nil {
		fmt.Println("Err rendering menu:" + err.Error())
	}
}

func ideaRemoveHandler(w http.ResponseWriter, r *http.Request) {
	ideaId := r.FormValue("ideaId")
	id, convErr := strconv.Atoi(ideaId)
	errString := ""
	if convErr != nil {
		id = 0
	}
	if id > 0 {
		models.DeleteIdea(id)
	} else {
		errString = "Invalid ID"
	}

	ideas, err := models.GetOpenIdeas()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	planData := PlanData{
		ErrorCode: errString,
		Ideas:     ideas,
	}
	buf := &bytes.Buffer{}
	err = planTpl.Execute(buf, planData)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_, err = buf.WriteTo(w)
	if err != nil {
		fmt.Println("Err rendering menu:" + err.Error())
	}
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
	article := ""
	title := ""
	var imgBytes []byte

	iLen, convErr := strconv.Atoi(length)
	if convErr != nil {
		iLen = 500
	}
	post := Post{
		Image:       imgBytes,
		Prompt:      promptEntry,
		ImagePrompt: imgPrompt,
		Length:      iLen,
	}
	useGpt4 := false
	if gpt4 == "true" {
		useGpt4 = true
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
		articleResp, err := openai.GenerateArticle(useGpt4, webPrompt.String(), viper.GetString("config.prompt.system-prompt"))
		if err != nil {
			post.Error = "Error generating article from OpenAI API: " + err.Error()
		}
		article = articleResp
		titleResp, err := openai.GenerateTitle(false, article, viper.GetString("config.prompt.title-prompt"), viper.GetString("config.prompt.system-prompt"))
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
		defer func() {
			response.Body.Close()
		}()
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

	updId, convErr := strconv.Atoi(ideaId)
	if convErr != nil {
		updId = 0
	}
	if updId > 0 {
		models.SetIdeaWritten(updId)
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
