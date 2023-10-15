package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	s "strings"

	"github.com/bitly/go-simplejson"
)

type TiktokVideo struct {
	url    string
	id     string
	author string
	desc   string
	width  int
	height int
}

type TiktokImages struct {
	urls      []string
	id        string
	music_url string
	author    string
	desc      string
}

/*
This function returns the HTTP Body reader for the raw mp4 link from a TiktokVideo struct
*/
func (t TiktokVideo) get_video_reader() io.Reader {
	res, err := http.Get(t.url)
	if err != nil {
		log.Printf("Couldn't download video %s\n", t.id)
	}

	return res.Body
}

/*
This function matches and gets the URL from a string using regexes
*/
func tiktok_is_valid(str string) string {
	reg, _ := regexp.Compile("https://vm.tiktok\\.com\\/{1}[a-zA-Z0-9]{9}[\\/]{0,1}|(https:\\/\\/www\\.tiktok\\.com\\/@[a-zA-Z0-9._]{0,32}\\/video\\/[0-9]{19}[?]{0,1}.{0,40})")

	return reg.FindString(str)

}

/*
This function checks if a TikTok URL is shortened
*/
func tiktok_is_shortened(url string) bool {
	log.Println("Checking if link is shortened")
	return s.Contains(url, "vm.tiktok.com")
}

/*
This function returns the link contained in the Location header of the HTTP response when querying a shortened link
*/
func tiktok_get_full_url(url string) string {
	log.Println("Link is shortened, getting full URL")
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	res, _ := client.Get(url)
	return res.Header.Get("Location")
}

/*
This function gets the aweme_id from a TikTok link using split
*/
func tiktok_extract_id(url string) string {
	log.Println("Extracting the ID from the URL")
	sl := s.Split(url, "?")
	sl = s.Split(sl[0], "/")

	return sl[len(sl)-1]
}

/*
This function returns a JSON object containing all the TikTok informations from a public TikTok endpoint
*/
func tiktok_extract_json(id string) (*simplejson.Json, bool, error) {

	api_endpoint := fmt.Sprintf("https://api16-normal-c-useast1a.tiktokv.com/aweme/v1/feed/?aweme_id=%s", id)

	log.Println("Querying the TikTok endpoint using the aweme ID: ", api_endpoint)
	res, err := http.Get(api_endpoint)
	if err != nil {
		return nil, false, err
	}

	buf := new(bytes.Buffer)
	buf.ReadFrom(res.Body)
	body_json := buf.Bytes()

	current_video, _ := simplejson.NewJson(body_json)
	current_video = current_video.Get("aweme_list").GetIndex(0)

	/*
		I set video as the default value because
		I think that all the TikTok JSONs that don't contain this field are videos
	*/

	is_image := current_video.Get("content_type").MustString("video") != "video"
	log.Println("Is this Tiktok an image? ", is_image)
	log.Println("content_type: ", current_video.Get("content_type").MustString())

	return current_video, is_image, nil
}

/*
This function creates a TikTokVideo struct and fills in all the fields based off the JSON output we get from tiktok_extract_json
*/
func tiktok_extract_video(current_video *simplejson.Json) TiktokVideo {

	/* Returns 3 strings containing the video link, description and author name */
	log.Println("Getting the video link from the JSON body")

	video_link := current_video.Get("video").Get("play_addr").Get("url_list").GetIndex(0).MustString()
	return TiktokVideo{
		url:    s.Split(video_link, "?")[0],
		id:     current_video.Get("aweme_id").MustString(),
		author: current_video.Get("author").Get("nickname").MustString(),
		desc:   current_video.Get("desc").MustString(),
		width:  current_video.Get("video").Get("play_addr").Get("width").MustInt(0),
		height: current_video.Get("video").Get("play_addr").Get("height").MustInt(0),
	}

}

/*
This function creates a TikTokImages struct and fills in all the fields based off the JSON output we get from tiktok_extract_json
*/
func tiktok_extract_images(current_video *simplejson.Json) TiktokImages {
	/* Returns a string array containing the images links, and 2 strings containing the description and author name */
	log.Println("Getting the images links from the JSON body")

	var images_links []string
	images := current_video.Get("image_post_info").Get("images")

	for i := 0; true; i += 1 {
		current_img := images.GetIndex(i).Get("display_image").Get("url_list").GetIndex(0).MustString("")

		if current_img == "" {
			break
		}

		log.Printf("Image link %d: %s\n", i, current_img)
		images_links = append(images_links, current_img)
	}

	return TiktokImages{
		urls:      images_links,
		id:        current_video.Get("aweme_id").MustString(),
		author:    current_video.Get("author").Get("nickname").MustString(),
		music_url: current_video.Get("music").Get("play_url").Get("uri").MustString(),
		desc:      current_video.Get("desc").MustString(),
	}

}
