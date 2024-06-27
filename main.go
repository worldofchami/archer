package main

import (
	// "context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"
	// "github.com/zmb3/spotify"
	// spotifyauth "github.com/zmb3/spotify/v2/auth"
	// "golang.org/x/oauth2"
)

type HTTPHandlerFunc = func(http.ResponseWriter, *http.Request)

func handleFatal(err error) {
	if err != nil {
		panic(err)
	}
}

type AuthResponse struct {
	AccessToken		string		`json:"access_token"`
	RefreshToken	string		`json:"refresh_token"`
}

func Login(clientId string) HTTPHandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		state := "ABCDEFGHIJKLMNOP"
		r.AddCookie(&http.Cookie{
			Name: "spotify_auth_state",
			Value: state,
		})

		scope := "user-read-private user-read-email"

		query_strs := fmt.Sprintf(
			"response_type=%s&client_id=%s&scope=%s&redirect_uri=%s&state=%s",
			"code",
			url.QueryEscape(clientId),
			url.QueryEscape(scope),
			url.QueryEscape("http://localhost:8080/callback"),
			url.QueryEscape(state),
		)

		http.Redirect(w, r, "https://accounts.spotify.com/authorize?" + query_strs, http.StatusFound)
	}
}

func Callback() HTTPHandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

	}
}

func main() {
	err := godotenv.Load(".env")
	handleFatal(err)

	clientId := os.Getenv("SPOTIFY_ID")
	// secret := os.Getenv("SPOTIFY_SECRET")

	httpClient := &http.Client{}

	res, err := httpClient.Do(&http.Request{
		Method: "GET",
		URL: &url.URL{
			Host: "localhost:8080",
			Path: "/test",
			Scheme: "http",
		},
	})
	handleFatal(err)

	defer res.Body.Close()

	token_info, err := io.ReadAll(res.Body)
	handleFatal(err)

	var authResponse AuthResponse

	json.Unmarshal(token_info, &authResponse)

	log.Println(authResponse)

	r := chi.NewRouter()
	r.Use(middleware.Logger)

	r.Get("/login", Login(clientId))

	// ctx := context.Background()

	// httpClient = spotifyauth.Authenticator{}.Client(ctx, &oauth2.Token{
	// 	AccessToken: "",
	// 	TokenType: "Bearer",
	// 	RefreshToken: "",
	// })
	
	// client := spotify.NewClient(httpClient)

	// // user, err := client.CurrentUser()
	// // handleFatal(err)

	// curr_playing, err := client.PlayerCurrentlyPlaying()
	// handleFatal(err)

	// log.Println(curr_playing.Item.Artists)
}