package unsplash

import (
	"errors"
	"github.com/hbagdi/go-unsplash/unsplash"
	"golang.org/x/oauth2"
	"io"
	"net/http"
	"strconv"
)

func GetImageBySearch(unsplashAccessKey string, searchString string) ([]byte, error) {
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
		return nil, err
	}
	imgUrl := ""
	for _, c := range *searchResults.Results {
		imgUrl = c.Urls.Regular.URL.String()
		break
	}
	response, err := http.Get(imgUrl)
	if err != nil {
		return nil, err
	}
	defer func() {
		response.Body.Close()
	}()
	if response.StatusCode != 200 {
		err = errors.New("Bad response code downloading image: " + strconv.Itoa(response.StatusCode))
		return nil, err
	}
	imgBytes, err = io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	return imgBytes, nil
}
