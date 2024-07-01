package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"

	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/joho/godotenv"
	"github.com/rivo/tview"
	"github.com/zmb3/spotify"
	spotifyauth "github.com/zmb3/spotify/v2/auth"
	"golang.org/x/oauth2"
)

// https://gist.github.com/sevkin/9798d67b2cb9d07cb05f89f14ba682f8
func open(url string) error {
    var cmd string
    var args []string

    switch runtime.GOOS {
    case "windows":
        cmd = "cmd"
        args = []string{"/c", "start"}
    case "darwin":
        cmd = "open"
    default: // "linux", "freebsd", "openbsd", "netbsd"
        cmd = "xdg-open"
    }
    args = append(args, url)
    return exec.Command(cmd, args...).Start()
}

type HTTPHandlerFunc = func(http.ResponseWriter, *http.Request)

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

		scope := "user-read-private user-read-email user-read-playback-state user-modify-playback-state user-read-currently-playing app-remote-control streaming playlist-read-private playlist-read-collaborative playlist-modify-private  user-read-playback-position user-top-read user-read-recently-played playlist-modify-public"

		query_strs := fmt.Sprintf(
			"response_type=%s&client_id=%s&scope=%s&redirect_uri=%s&state=%s",
			"code",
			url.QueryEscape(clientId),
			url.QueryEscape(scope),
			url.QueryEscape("http://localhost:8888/callback"),
			url.QueryEscape(state),
		)

		// http.Redirect(w, r, "https://accounts.spotify.com/authorize?" + query_strs, http.StatusFound)
		open("https://accounts.spotify.com/authorize?" + query_strs)
	}
}

func Callback(clientId, clientSecret string, authResponse *chan AuthResponse) HTTPHandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		state := r.URL.Query().Get("code")
		// stored_state, err := r.Cookie("stateKey")

		if state == "" { // || err != nil || state != stored_state.Value {
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

			// query_url, _ := url.Parse("https://accounts.spotify.com/api/token")

			formData := url.Values{}
			formData.Set("code", code)
			formData.Set("redirect_uri", "http://localhost:8888/callback")
			formData.Set("grant_type", "authorization_code")
			// post_req := &http.Request{
			// 	Method: "POST",
			// 	URL: query_url,
			// 	Form: map[string][]string{
			// 		"code": { code },
			// 		"redirect_uri": { "http://localhost:8888/callback" },
			// 		"grant_type": { "authorization_code" },
			// 	},
			// 	Header: map[string][]string{
			// 		"Content-Type": { "application/x-www-form-urlencoded" },
			// 		"Authorization": { buffer.String() },
			// 	},
			// }

			post_req, _ := http.NewRequest("POST", "https://accounts.spotify.com/api/token", strings.NewReader(formData.Encode()))
			post_req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
			post_req.Header.Add("Authorization", fmt.Sprintf(
				"Basic %s",
				base64.RawStdEncoding.EncodeToString([]byte(clientId + ":" + clientSecret)),
			))

			res, err := http_client.Do(post_req)
			handleGraceful(err)

			defer res.Body.Close()

			if res.StatusCode == 200 {
				token_info, err := io.ReadAll(res.Body)
				handleGraceful(err)

				var ar AuthResponse
				err = json.Unmarshal(token_info, &ar)
				handleFatal(err)

				*authResponse<-ar

				// get_url, _ := url.Parse("https://api.spotify.com/v1/me")

				// get_req := &http.Request{
				// 	Method: "GET",
				// 	URL: get_url,
				// 	Header: map[string][]string{
				// 		"Authorization": { fmt.Sprintf("Bearer %s", authResponse.AccessToken) },
				// 	},
				// }

				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(&ar)
			} else {
				body, err := io.ReadAll(res.Body)
				handleGraceful(err)

				log.Print(string(body))
				w.WriteHeader(http.StatusUnauthorized)
			}
		}
	}
}

func startServer(clientId, clientSecret *string, authResponse *chan AuthResponse) {
	r := chi.NewRouter()
	r.Use(middleware.Logger)

	r.Get("/login", Login(*clientId))
	r.Get("/callback", Callback(*clientId, *clientSecret, authResponse))
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
        fmt.Fprintln(w, "Hello, world!")
    })
	
	log.Print("Listening on 8888")
	http.ListenAndServe(":8888", r)
}

func main() {
	err := godotenv.Load(".env")
	handleFatal(err)

	clientId := os.Getenv("SPOTIFY_ID")
	clientSecret := os.Getenv("SPOTIFY_SECRET")

	authResponse := make(chan AuthResponse, 2)

	go startServer(&clientId, &clientSecret, &authResponse)

	url, _ := url.Parse("http://localhost:8888/login")

	res, err := (&http.Client{}).Do(&http.Request{
		Method: "GET",
		URL: url,
	})
	handleFatal(err)

	defer res.Body.Close()

	token := <-authResponse

	fmt.Println(token)

	// Read empty value from chan to close program
	defer func() {
		authResponse<-AuthResponse{}
		<-authResponse
	}()

	ctx := context.Background()

	http_client := spotifyauth.Authenticator{}.Client(ctx, &oauth2.Token{
		AccessToken: token.AccessToken,
		TokenType: "Bearer",
		RefreshToken: token.RefreshToken,
	})
	
	client := spotify.NewClient(http_client)

	curr_playing, err := client.PlayerCurrentlyPlaying()
	handleFatal(err)

	artists := curr_playing.Item.Artists
	var artists_str string

	for _, artist := range(artists[:len(artists)-1]) {
		artists_str += artist.Name + ", "
	}

	artists_str += artists[len(artists)-1].Name

	song_name := curr_playing.Item.Name

	// res, err = http.Get("https://i.scdn.co/image/ab67616d0000b2732b9aca3204e667980ce6a939")
    // handleFatal(err)

    // defer res.Body.Close()

	// cover, err := jpeg.Decode(res.Body)
	// handleFatal(err)

	// song_name := "King James"
	// artists_str := "Anderson .Paak"

	app := tview.NewApplication()
	flex := tview.NewFlex().
		AddItem(
			tview.NewFlex().SetDirection(tview.FlexColumnCSS).
			AddItem(
				tview.NewFlex().SetDirection(tview.FlexColumnCSS).
				AddItem(
					tview.NewTextView().SetText("Playlist").SetTextAlign(1).SetTextColor(tcell.Color(tcell.NewRGBColor(238, 255, 0))), 2, 1, false,
				),
				1, 1, false,
			).
			AddItem(
				tview.NewImage().SetAspectRatio(1).SetBorder(true),
				0, 1, false,
			).
			AddItem(
				tview.NewFlex().SetDirection(tview.FlexColumnCSS).
				AddItem(
					tview.NewTextView().SetText(song_name).SetTextAlign(1).SetTextColor(tcell.Color(tcell.NewRGBColor(238, 255, 0))),
					2, 2, false,
				).
				AddItem(
					tview.NewTextView().SetText(artists_str).SetTextAlign(1),
					1, 1, false,
				),
				3, 1, false,
			).
			AddItem(
				tview.NewFlex().SetDirection(tview.FlexRowCSS).
				AddItem(
					tview.NewButton("<").SetMouseCapture(func(action tview.MouseAction, event *tcell.EventMouse) (tview.MouseAction, *tcell.EventMouse) {
						if action == tview.MouseLeftClick {
							err := client.Previous()
							handleGraceful(err)
						}
						
						return action, event
					}),
					5, 1, false,
				).
				AddItem(
					tview.NewButton("P").SetMouseCapture(func(action tview.MouseAction, event *tcell.EventMouse) (tview.MouseAction, *tcell.EventMouse) {
						if action == tview.MouseLeftClick {
							err := client.Pause()
							handleGraceful(err)
						}
						
						return action, event
					}),
					5, 1, false,
				).
				AddItem(
					tview.NewButton(">").SetMouseCapture(func(action tview.MouseAction, event *tcell.EventMouse) (tview.MouseAction, *tcell.EventMouse) {
						if action == tview.MouseLeftClick {
							err := client.Next()
							handleGraceful(err)
						}
						
						return action, event
					}),
					5, 1, false,
				),
				1, 0, false,
			),
		0, 1, false)
			
	if err := app.SetRoot(flex, true).SetFocus(flex).Run(); err != nil {
		panic(err)
	}
}