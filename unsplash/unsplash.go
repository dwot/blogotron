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
	handleError(err)
	imgUrl := ""
	for _, c := range *searchResults.Results {
		imgUrl = c.Urls.Regular.URL.String()
		break
	}
	response, err := http.Get(imgUrl)

	handleError(err)
	defer func() {
		response.Body.Close()
	}()
	if response.StatusCode != 200 {
		fmt.Println("Bad response code downloading image: " + strconv.Itoa(response.StatusCode))
	}
	imgBytes, err = io.ReadAll(response.Body)
	handleError(err)
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
	handleError(err)
	imgUrl := ""
	for _, c := range *randomPhoto {
		imgUrl = c.Urls.Regular.URL.String()
	}
	response, err := http.Get(imgUrl)

	handleError(err)
	defer func() {
		response.Body.Close()
	}()
	if response.StatusCode != 200 {
		fmt.Println("Bad response code downloading image: " + strconv.Itoa(response.StatusCode))
	}
	imgBytes, err = io.ReadAll(response.Body)
	handleError(err)
	return imgBytes
}

func handleError(err error) {
	if err != nil {
		fmt.Println("Error: ", err.Error())
	}
}
