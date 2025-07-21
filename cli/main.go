package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image/jpeg"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type Player struct {
	SongName		string 			`json:"song_name"`
	Artists			string			`json:"artists"`
	AlbumCoverURL	string			`json:"album_cover_url"`
	IsPlaying		bool			`json:"is_playing"`
	IsPaused		bool			`json:"is_paused"`
	PlaylistName	string			`json:"playlist_name"`
}

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

func log(message string) {
	os.WriteFile("./logs.txt", []byte(fmt.Sprintf(
		"LOG: %s - %s\n",
		time.Now().Format(time.ANSIC),
		message,
	)), 0644)
}

var (
	FOCUS_STYLE=tcell.Style{}.Background(tcell.Color(tcell.NewRGBColor(238, 255, 0)))
	BLUR_STYLE=tcell.Style{}.Background(tcell.Color(tcell.NewRGBColor(0, 0, 0)))
)

type AppInterface struct {
	SongNameTextView *tview.TextView
	ArtistsTextView *tview.TextView
	AlbumCover *tview.Image
}

type ApiClient struct {
	app_interface *AppInterface
	server_url string
	auth_token string
}

func (api_client *ApiClient) GetPlayerState() (*Player, error) {
	http_client := &http.Client{}

	req, err := http.NewRequest("GET", api_client.server_url + "/player", nil)
	handleGraceful(err)

	req.Header.Set("Authorization", "Bearer " + api_client.auth_token)

	res, err := http_client.Do(req)
	handleGraceful(err)

	defer res.Body.Close()

	var player Player
	json.NewDecoder(res.Body).Decode(&player)

	return &player, nil
}

func (api_client *ApiClient) refreshPlayer(player *Player) (*Player, error) {
	bin_data, err := getImageBytes(player.AlbumCoverURL)
	handleGraceful(err)

	photo, _ := jpeg.Decode(bytes.NewReader(bin_data))
	api_client.app_interface.AlbumCover.SetImage(photo).SetColors(tview.TrueColor).SetAspectRatio(1)
	
	updateTextView(player.SongName, api_client.app_interface.SongNameTextView)
	updateTextView(player.Artists, api_client.app_interface.ArtistsTextView)

	return player, nil
}

func (api_client *ApiClient) pause() {
	http_client := &http.Client{}

	req, err := http.NewRequest("POST", api_client.server_url + "/pause", nil)
	handleGraceful(err)

	req.Header.Set("Authorization", "Bearer " + api_client.auth_token)

	res, err := http_client.Do(req)
	handleGraceful(err)

	defer res.Body.Close()

	var player Player
	json.NewDecoder(res.Body).Decode(&player)

	api_client.refreshPlayer(&player)
}

func (api_client *ApiClient) prev() {
	http_client := &http.Client{}

	req, err := http.NewRequest("POST", api_client.server_url + "/prev", nil)
	handleGraceful(err)

	req.Header.Set("Authorization", "Bearer " + api_client.auth_token)

	res, err := http_client.Do(req)
	handleGraceful(err)

	defer res.Body.Close()

	var player Player
	json.NewDecoder(res.Body).Decode(&player)

	api_client.refreshPlayer(&player)
}

func (api_client *ApiClient) next() {
	http_client := &http.Client{}

	req, err := http.NewRequest("POST", api_client.server_url + "/next", nil)
	handleGraceful(err)

	req.Header.Set("Authorization", "Bearer " + api_client.auth_token)

	res, err := http_client.Do(req)
	handleGraceful(err)

	defer res.Body.Close()

	var player Player
	json.NewDecoder(res.Body).Decode(&player)

	api_client.refreshPlayer(&player)
}

func updateTextView(text string, tv *tview.TextView) {
	w := tv.BatchWriter()
	defer w.Close()

	w.Clear()
	w.Write([]byte(text))
}

func getImageBytes(url string) ([]byte, error) {
	res, err := http.Get(url)
	handleGraceful(err)

	defer res.Body.Close()
	bin_data, err := io.ReadAll(res.Body)
	handleGraceful(err)

	return bin_data, nil
}

func writeToken(token string) {
	os.WriteFile("./token", []byte(token), 0644)
}

func pollPlayerState(api_client *ApiClient, app *tview.Application) {
	for {
		player, err := api_client.GetPlayerState()
		handleGraceful(err)
		
		app.QueueUpdateDraw(func() {
			api_client.refreshPlayer(player)
		})
		
		time.Sleep(2 * time.Second)
	}
}

func main() {
	app_url := os.Getenv("APP_URL")

	if app_url == "" {
		app_url = "http://localhost:3000"
	}

	var auth_token string

	token_file, _ := os.ReadFile("./token")
	if len(token_file) == 0 {
		fmt.Println("Please enter the ID you see in your browser (or visit " + app_url + "/setup):")
		fmt.Scanln(&auth_token)
		writeToken(auth_token)
	} else {
		auth_token = string(token_file)
	}

	server_url := os.Getenv("SERVER_URL")

	if server_url == "" {
		server_url = "http://localhost:8888"
	}

	api_client := &ApiClient{
		app_interface: &AppInterface{},
		server_url: server_url,
		auth_token: auth_token,
	}
	app := tview.NewApplication()

	player, err := api_client.GetPlayerState()
	handleFatal(err)

	// go pollPlayerState(api_client, app)

	if player.SongName == "" || !player.IsPlaying {
		text := tview.NewTextView().
			SetText("it's awfully quiet here. i can hear the voices in my head. please play something").
			SetTextColor(tcell.Color(tcell.NewRGBColor(238, 255, 0))).
			SetTextAlign(tview.AlignCenter)

		if err := app.SetRoot(text, true).Run(); err != nil {
			panic(err)
		}
		return
	}

	artists_str := player.Artists
	song_name := player.SongName
	album_cover_url := player.AlbumCoverURL
	playlist_name := player.PlaylistName

	bin_data, err := getImageBytes(album_cover_url)
	handleGraceful(err)

	cover := tview.NewImage()
	photo, _ := jpeg.Decode(bytes.NewReader(bin_data))
	cover.SetImage(photo).SetColors(tview.TrueColor).SetAspectRatio(0.5)

	const (
		NONE=0
		PREV=1
		PAUSE=2
		NEXT=3
	)

	focused_control := NONE

	buttons := []*tview.Button{
		tview.NewButton("|<"),
		tview.NewButton("||"),
		tview.NewButton(">|"),
	}

	song_name_text_view := tview.NewTextView().SetText(song_name).SetTextAlign(1).SetTextColor(tcell.Color(tcell.NewRGBColor(238, 255, 0)))
	artists_text_view := tview.NewTextView().SetText(artists_str).SetTextAlign(1)

	playlist_name_text_view := tview.NewTextView().SetText(playlist_name).SetTextAlign(1).SetTextColor(tcell.Color(tcell.NewRGBColor(238, 255, 0)))

	api_client.app_interface.SongNameTextView = song_name_text_view
	api_client.app_interface.ArtistsTextView = artists_text_view
	api_client.app_interface.AlbumCover = cover

	control_box := tview.NewBox().SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		key := event.Key()

		if key == tcell.KeyTAB {
			// Focus on next control
			focused_control = (focused_control + 1) % 4

			// TODO: try make more efficient
			for _, button := range buttons {
				button.SetStyle(BLUR_STYLE)
			}

			if focused_control > 0 {
				buttons[focused_control-1].SetStyle(FOCUS_STYLE)
			}
		} else if key == tcell.KeyEnter {
			// Do action for this key, set it to focused
			switch focused_control {
				case NEXT: {
					api_client.next()
				}
				case PREV: {
					api_client.prev()
				}
				case PAUSE: {
					api_client.pause()
				}
			}
		}
		
		return event
	})

	controls := tview.NewFlex().SetDirection(tview.FlexRowCSS).
	AddItem(
		buttons[0],
		5, 1, false,
	).
	AddItem(
		buttons[1],
		5, 1, false,
	).
	AddItem(
		buttons[2],
		5, 1, false,
	).AddItem(control_box, 0,0,false)

	flex := tview.NewFlex().
		AddItem(
			tview.NewFlex().SetDirection(tview.FlexColumnCSS).
			AddItem(
				tview.NewFlex().SetDirection(tview.FlexColumnCSS).
				AddItem(
					playlist_name_text_view, 2, 1, false,
				),
				1, 1, false,
			).
			AddItem(
				cover,
				20, 3, false,
			).
			AddItem(
				tview.NewFlex(),
				1, 1, false,
			).
			AddItem(
				tview.NewFlex().SetDirection(tview.FlexRowCSS).
				AddItem(
					// STOPPED HERE
					tview.NewGrid(),
					2, 1, false,
				),
				4, 1, false,
			).
			AddItem(
				tview.NewFlex().SetDirection(tview.FlexColumnCSS).
				AddItem(
					song_name_text_view,
					2, 2, false,
				).
				AddItem(
					artists_text_view,
					1, 1, false,
				),
				3, 1, false,
			).
			AddItem(
				controls,
				1, 0, false,
			),
		0, 1, false)
			
	if err := app.SetRoot(flex, true).SetFocus(control_box).Run(); err != nil {
		panic(err)
	}
}