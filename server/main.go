package main

import (
	"context"
	// "crypto/rand"
	"encoding/base64"
	"encoding/json"
	// "errors"
	"fmt"
	"io"
	"log"
	// "math/big"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"
	"github.com/zmb3/spotify"
	spotifyauth "github.com/zmb3/spotify/v2/auth"
	"golang.org/x/oauth2"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	// "github.com/golang-jwt/jwt/v5"
)

type HTTPHandlerFunc = func(http.ResponseWriter, *http.Request)

type AuthResponse struct {
	AccessToken		string		`json:"access_token"`
	RefreshToken	string		`json:"refresh_token"`
}

type SpotifyUser struct {
	DisplayName	string		`json:"display_name"`
	Email		string		`json:"email"`
}

type User struct {
	ID		string			`json:"id"`
	Email	string			`json:"email"`
	AccessToken	string		`json:"access_token"`
	RefreshToken	string	`json:"refresh_token"`
}

// func randomStr(length int) string {
//     letters := "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
//     result := make([]byte, length)

//     for i := range result {
//         num, _ := rand.Int(rand.Reader, big.NewInt(int64(len(letters))))

//         result[i] = letters[num.Int64()]
//     }

//     return string(result)
// }

func handleFatal(err error) {
	if err != nil {
		os.WriteFile("./logs.txt", []byte(fmt.Sprintf(
			"TERMINATED: %s - %s\n",
			time.Now().Format(time.ANSIC),
			err,
		)), 0644)

		panic(err)
	}
}

func handleGraceful(err error) {
	if err != nil {
		os.WriteFile("./logs.txt", []byte(fmt.Sprintf(
			"ERROR: %s - %s\n",
			time.Now().Format(time.ANSIC),
			err,
		)), 0644)
	}
}

func LoginWithSpotify(db *gorm.DB, clientId string) HTTPHandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user_id := r.URL.Query().Get("user_id")

		var user User

		db.Raw("SELECT * FROM spotify_tokens WHERE user_id = ?", user_id).Scan(&user)

		if user.ID == "" {
			// User's tokens don't exist, link them
			state := user_id
			r.AddCookie(&http.Cookie{
				Name: "spotify_auth_state",
				Value: state,
			})
	
			scope := "user-read-private user-read-email user-read-playback-state user-modify-playback-state user-read-currently-playing app-remote-control streaming playlist-read-private playlist-read-collaborative playlist-modify-private  user-read-playback-position user-top-read user-read-recently-played playlist-modify-public"
	
			query_strs := fmt.Sprintf(
				"response_type=%s&client_id=%s&scope=%s&redirect_uri=%s&state=%s",
				"code",
				url.QueryEscape(clientId),
				url.QueryEscape(scope),
				url.QueryEscape(os.Getenv("SERVER_URL") + "/callback"),
				url.QueryEscape(state),
			)

			log.Print(query_strs)
	
			http.Redirect(w, r, "https://accounts.spotify.com/authorize?" + query_strs, http.StatusFound)
		} else {
			// User already linked Spotify
		}
	}
}

func Callback(db *gorm.DB, clientId, clientSecret string) HTTPHandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		state := r.URL.Query().Get("state")
		// stored_state, err := r.Cookie("stateKey")

		if false { //err != nil || state != stored_state.Value {
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

			formData := url.Values{}
			formData.Set("code", code)
			formData.Set("redirect_uri", os.Getenv("SERVER_URL") + "/callback")
			formData.Set("grant_type", "authorization_code")

			post_req, _ := http.NewRequest("POST", "https://accounts.spotify.com/api/token", strings.NewReader(formData.Encode()))
			post_req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
			post_req.Header.Add("Authorization", fmt.Sprintf(
				"Basic %s",
				base64.RawStdEncoding.EncodeToString([]byte(clientId + ":" + clientSecret)),
			))

			res, err := http_client.Do(post_req)
			handleGraceful(err)

			defer res.Body.Close()

			if res.StatusCode != 200 {
				body, err := io.ReadAll(res.Body)
				handleGraceful(err)

				log.Print(string(body))
				w.WriteHeader(res.StatusCode)
				return
			}

			token_info, err := io.ReadAll(res.Body)
			handleGraceful(err)

			var ar AuthResponse
			err = json.Unmarshal(token_info, &ar)
			handleFatal(err)

			get_url, _ := url.Parse("https://api.spotify.com/v1/me")

			get_req := &http.Request{
				Method: "GET",
				URL: get_url,
				Header: map[string][]string{
					"Authorization": { fmt.Sprintf("Bearer %s", ar.AccessToken) },
				},
			}

			res, err = http_client.Do(get_req)
			handleGraceful(err)

			defer res.Body.Close()

			spotify_user, err := io.ReadAll(res.Body)
			handleGraceful(err)

			var su SpotifyUser
			err = json.Unmarshal(spotify_user, &su)
			handleFatal(err)

			db.Exec(`
				INSERT INTO spotify_tokens (user_id, access_token, refresh_token, email, display_name) VALUES (?, ?, ?, ?, ?)
				ON CONFLICT (user_id) DO UPDATE SET access_token = ?, refresh_token = ?
			`,
			state, ar.AccessToken, ar.RefreshToken, su.Email, su.DisplayName,
			ar.AccessToken, ar.RefreshToken)

			http.Redirect(w, r, fmt.Sprintf("%s/setup", os.Getenv("APP_URL")), http.StatusFound)
		}
	}
}

func fetchPlayerState(client *spotify.Client) (*Player, error) {
	curr_playing, err := client.PlayerCurrentlyPlaying()
	handleGraceful(err)

	artists_str := concatArtists(curr_playing.Item.Artists)
	song_name := curr_playing.Item.Name
	album_cover_url := curr_playing.Item.Album.Images[0].URL
	playlist_name := curr_playing.Item.Album.Name

	player := Player{
		SongName: song_name,
		Artists: artists_str,
		AlbumCoverURL: album_cover_url,
		IsPlaying: curr_playing.Playing,
		PlaylistName: playlist_name,
	}

	return &player, nil
}

func GetPlayerState(db *gorm.DB, clientId string) HTTPHandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := context.Background()
		user, err := authenticate(db, r)
		handleFatal(err)

		http_client := spotifyauth.Authenticator{}.Client(ctx, &oauth2.Token{
			AccessToken: user.AccessToken,
			RefreshToken: user.RefreshToken,
		})

		client := spotify.NewClient(http_client)

		player, err := fetchPlayerState(&client)
		handleGraceful(err)

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(&player)
	}
}

func Play(db *gorm.DB, clientId string) HTTPHandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := context.Background()
		user, err := authenticate(db, r)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		http_client := spotifyauth.Authenticator{}.Client(ctx, &oauth2.Token{
			AccessToken: user.AccessToken,
			RefreshToken: user.RefreshToken,
		})

		client := spotify.NewClient(http_client)

		err = client.Play()
		handleFatal(err)

		player, err := fetchPlayerState(&client)
		handleGraceful(err)

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(&player)
	}
}

func Pause(db *gorm.DB, clientId string) HTTPHandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := context.Background()
		user, err := authenticate(db, r)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		http_client := spotifyauth.Authenticator{}.Client(ctx, &oauth2.Token{
			AccessToken: user.AccessToken,
			RefreshToken: user.RefreshToken,
		})

		client := spotify.NewClient(http_client)

		player, err := fetchPlayerState(&client)
		handleGraceful(err)

		if player.IsPlaying {
			err = client.Pause()
			handleGraceful(err)
		} else {
			err = client.Play()
			handleGraceful(err)
		}
		
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(&player)
	}
}

func Next(db *gorm.DB, clientId string) HTTPHandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := context.Background()
		user, err := authenticate(db, r)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		http_client := spotifyauth.Authenticator{}.Client(ctx, &oauth2.Token{
			AccessToken: user.AccessToken,
			RefreshToken: user.RefreshToken,
		})

		client := spotify.NewClient(http_client)

		err = client.Next()
		handleGraceful(err)

		player, err := fetchPlayerState(&client)
		handleGraceful(err)

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(&player)
	}
}

func Prev(db *gorm.DB, clientId string) HTTPHandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := context.Background()
		user, err := authenticate(db, r)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		http_client := spotifyauth.Authenticator{}.Client(ctx, &oauth2.Token{
			AccessToken: user.AccessToken,
			RefreshToken: user.RefreshToken,
		})

		client := spotify.NewClient(http_client)

		err = client.Previous()
		handleGraceful(err)

		player, err := fetchPlayerState(&client)
		handleGraceful(err)

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(&player)
	}
}

func concatArtists(artists []spotify.SimpleArtist) string {
	artists_str := ""

	for _, artist := range(artists[:len(artists)-1]) {
		artists_str += artist.Name + ", "
	}

	artists_str += artists[len(artists)-1].Name

	return artists_str
}

func authenticate(db *gorm.DB, r *http.Request) (User, error) {
	var user User
	auth_header := r.Header.Get("Authorization")
	log.Print(auth_header)
	auth_token := strings.Split(auth_header, " ")[1]

	db.Raw("SELECT * FROM spotify_tokens WHERE user_id = ?", auth_token).Scan(&user)

	return user, nil
	
	// token, _, err := new(jwt.Parser).ParseUnverified(auth_token, jwt.MapClaims{})
    // if err != nil {
    //     fmt.Println(err)
    // }

    // if claims, ok := token.Claims.(jwt.MapClaims); ok {
	// 	var user User
	// 	db.Raw("SELECT * FROM spotify_tokens WHERE user_id = ?", claims["sub"]).Scan(&user)

    //     return user, nil
    // } else {
    //     return User{
	// 		Email: "",
	// 		AccessToken: "",
	// 		RefreshToken: "",
	// 	}, errors.New("not authenticated")
    // }
}

type Player struct {
	SongName		string 			`json:"song_name"`
	Artists			string			`json:"artists"`
	AlbumCoverURL	string			`json:"album_cover_url"`
	IsPlaying		bool			`json:"is_playing"`
	IsPaused		bool			`json:"is_paused"`
	PlaylistName	string			`json:"playlist_name"`
}

const MAX_CONN_COUNT = 40

func main() {
	err := godotenv.Load(".env")
	handleFatal(err)

	clientId := os.Getenv("SPOTIFY_ID")
	clientSecret := os.Getenv("SPOTIFY_SECRET")

	dsn := os.Getenv("CONN_STR")

	db, err := gorm.Open(postgres.New(postgres.Config{
		DSN: dsn,
		PreferSimpleProtocol: true,
	}), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}

	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(MAX_CONN_COUNT)

	r := chi.NewRouter()
	r.Use(middleware.Logger)

	r.Get("/login", LoginWithSpotify(db, clientId))
	r.Get("/callback", Callback(db, clientId, clientSecret))
	r.Get("/player", GetPlayerState(db, clientId))
	r.Post("/play", Play(db, clientId))
	r.Post("/pause", Pause(db, clientId))
	r.Post("/next", Next(db, clientId))
	r.Post("/prev", Prev(db, clientId))
	log.Print("Listening on 8888")
	http.ListenAndServe(":8888", r)
}