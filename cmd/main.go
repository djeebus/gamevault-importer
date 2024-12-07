package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/pkg/errors"
)

type file struct {
	ManualURL string `json:"manualUrl"`
	Name      string `json:"name"`
	Version   string `json:"version"`
	Date      string `json:"date"`
	Size      string `json:"size"`
}

type gameData struct {
	Downloads        [][]any
	Title            string
	ReleaseTimestamp int64
}

func Cmd() error {
	var client http.Client

	cookies := map[string]string{
		"gog-al": os.Getenv("AUTH_GOG_AL"),
		"gog_lc": os.Getenv("AUTH_GOG_LC"),
		"gog_us": os.Getenv("AUTH_GOG_US"),
	}

	gameIDs, err := getGameIDs(cookies, client)
	if err != nil {
		return err
	}

	for _, gameID := range gameIDs {
		files, err := getDownloads(gameID, cookies, client)
		if err != nil {
			return errors.Wrapf(err, "failed to get game data for %s", gameID)
		}

		fmt.Printf("Found %d files\n", len(files))
	}

	return nil
}

func findLatestDownload(language, platform string, data gameData) (string, []any, error) {
	for _, installers := range data.Downloads {

		if len(installers) != 2 {
			println("installers is not a 2-length tuple")
		}

	}
	return "", nil, nil
}

func getDownloads(gameID string, cookies map[string]string, client http.Client) ([]file, error) {
	// GET https://www.gog.com/account/gameDetails/$id.json
	url := fmt.Sprintf("https://www.gog.com/account/gameDetails/%s.json", gameID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create second request")
	}
	addCookies(req, cookies)

	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch second request")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read second request")
	}

	if err = os.WriteFile(fmt.Sprintf("%s.json", gameID), body, 0644); err != nil {
		return nil, errors.Wrap(err, "failed to write second request")
	}

	var data gameData
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, errors.Wrap(err, "failed to decode second request")
	}

	for _, download := range data.Downloads {
		if len(download) != 2 {
			return nil, errors.New("downloads is not a 2-length tuple")
		}

		if download[0] != "English" {
			continue
		}

		oses, ok := download[1].(map[string]any)
		if !ok {
			return nil, errors.New("downloads[1] is not a map[string]any")
		}

		windows, ok := oses["windows"]
		if !ok {
			return nil, errors.New("downloads[1] windows is not found")
		}

		data, err := json.Marshal(windows)
		if err != nil {
			return nil, errors.Wrap(err, "failed to encode windows files")
		}

		var files []file
		if err = json.Unmarshal(data, &files); err != nil {
			return nil, errors.Wrap(err, "failed to decode windows files")
		}

		return files, nil
	}

	return nil, errors.New("failed to find a single download")
}

func getGameIDs(cookies map[string]string, client http.Client) ([]string, error) {
	req, err := http.NewRequest("GET", "https://menu.gog.com/v1/account/licences", nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create first request")
	}
	addCookies(req, cookies)

	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute first request")
	}

	var gameIDs []string
	if err := json.NewDecoder(resp.Body).Decode(&gameIDs); err != nil {
		return nil, errors.Wrap(err, "failed to decode response body")
	}
	return gameIDs, nil
}

func addCookies(req *http.Request, cookies map[string]string) {
	for name, value := range cookies {
		req.AddCookie(&http.Cookie{Name: name, Value: value})
	}
}
