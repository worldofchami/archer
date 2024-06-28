package main

import (
	// "context"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

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

func handleGraceful(err error) {
	if err != nil {
		log.Print(err)
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

func Callback(clientId, clientSecret string) HTTPHandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		state := r.URL.Query().Get("code")
		stored_state, err := r.Cookie("stateKey")

		if state == "" || err != nil || state != stored_state.Value {
			http.Redirect(w, r, "/error?error=state_mismatch", http.StatusFound)
		} else {
			http.SetCookie(w, &http.Cookie{
				Name:     "stateKey",
				Value:    "",
				Path:     "/",
				Expires: time.Unix(0, 0),
				HttpOnly: true,
			})

			http_client := &http.Client{}

			query_url, _ := url.Parse("https://accounts.spotify.com/api/token")
			
			var buffer bytes.Buffer
			buffer.WriteString(fmt.Sprintf(
				"Basic %s:%s",
				clientId,
				clientSecret,
			))

			post_req := &http.Request{
				Method: "POST",
				URL: query_url,
				Form: map[string][]string{
					"code": { code },
					"redirect_url": { "http://localhost:8080/callback" },
					"grant_type": { "authorization_code" },
				},
				Header: map[string][]string{
					"Content-Type": { "application/x-www-form-urlencoded" },
					"Authorization": { buffer.String() },
				},
			}

			res, err := http_client.Do(post_req)
			handleGraceful(err)

			defer res.Body.Close()

			if res.StatusCode == 200 {
				token_info, err := io.ReadAll(res.Body)
				handleGraceful(err)

				var authResponse AuthResponse

				json.Unmarshal(token_info, &authResponse)

				// Stopped here
				// get_req := &http.Request{

				// }
			}
		}
	}
}

func main() {
	err := godotenv.Load(".env")
	handleFatal(err)

	clientId := os.Getenv("SPOTIFY_ID")
	clientSecret := os.Getenv("SPOTIFY_SECRET")

	http_client := &http.Client{}

	url, _ := url.Parse("http://localhost:8080/test")
	res, err := http_client.Do(&http.Request{
		Method: "GET",
		URL: url,
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
	r.Get("/callback", Callback(clientId, clientSecret))

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