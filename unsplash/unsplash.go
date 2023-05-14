package unsplash

import (
	"fmt"
	"github.com/hbagdi/go-unsplash/unsplash"
	"golang.org/x/oauth2"
	"io"
	"net/http"
	"os"
	"strconv"
)

func GetImageBySearch(searchString string) []byte {
	unsplashAccessKey := os.Getenv("UNSPLASH_ACCESS_KEY")
	imgBytes := []byte{}
	ts := oauth2.StaticTokenSource(
		// note Client-ID in front of the access token
		&oauth2.Token{AccessToken: "Client-ID " + unsplashAccessKey},
	)
	client := oauth2.NewClient(oauth2.NoContext, ts)
	unClient := unsplash.New(client)

	opt := unsplash.SearchOpt{}
	opt.Query = searchString
	searchResults, _, err := unClient.Search.Photos(&opt)
	if err != nil {
		fmt.Println("Error getting random photo: " + err.Error())
	}
	imgUrl := ""
	for _, c := range *searchResults.Results {
		imgUrl = c.Urls.Regular.URL.String()
		break
	}
	response, err := http.Get(imgUrl)

	if err != nil {
		fmt.Println("Error downloading image: " + err.Error())
	}
	defer func() {
		response.Body.Close()
	}()
	if response.StatusCode != 200 {
		fmt.Println("Bad response code downloading image: " + strconv.Itoa(response.StatusCode))
	}
	imgBytes, err = io.ReadAll(response.Body)
	if err != nil {
		fmt.Println("Error reading image bytes: " + err.Error())
	}
	if err != nil {
		fmt.Println("Error getting random photo: " + err.Error())
	}
	return imgBytes
}

func GetRandomImage() []byte {
	unsplashAccessKey := os.Getenv("UNSPLASH_ACCESS_KEY")
	imgBytes := []byte{}
	ts := oauth2.StaticTokenSource(
		// note Client-ID in front of the access token
		&oauth2.Token{AccessToken: "Client-ID " + unsplashAccessKey},
	)
	client := oauth2.NewClient(oauth2.NoContext, ts)
	unClient := unsplash.New(client)
	// requests can be now made to the API

	randomPhoto, _, err := unClient.Photos.Random(nil)
	if err != nil {
		fmt.Println("Error getting random photo: " + err.Error())
	}
	imgUrl := ""
	for _, c := range *randomPhoto {
		imgUrl = c.Urls.Regular.URL.String()
	}
	response, err := http.Get(imgUrl)

	if err != nil {
		fmt.Println("Error downloading image: " + err.Error())
	}
	defer func() {
		response.Body.Close()
	}()
	if response.StatusCode != 200 {
		fmt.Println("Bad response code downloading image: " + strconv.Itoa(response.StatusCode))
	}
	imgBytes, err = io.ReadAll(response.Body)
	if err != nil {
		fmt.Println("Error reading image bytes: " + err.Error())
	}
	if err != nil {
		fmt.Println("Error getting random photo: " + err.Error())
	}
	return imgBytes
}
