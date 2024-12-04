package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/pkg/errors"
)

type installers struct {
	OS, Path string
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
		data, err := getGameData(gameID, cookies, client)
		if err != nil {
			return errors.Wrapf(err, "failed to get game data for %s", gameID)
		}

		_, _, err = findLatestDownload("English", "windows", data)
		return err
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

func getGameData(gameID string, cookies map[string]string, client http.Client) (gameData, error) {
	// GET https://www.gog.com/account/gameDetails/$id.json
	url := fmt.Sprintf("https://www.gog.com/account/gameDetails/%s.json", gameID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return gameData{}, errors.Wrap(err, "failed to create second request")
	}
	addCookies(req, cookies)

	resp, err := client.Do(req)
	if err != nil {
		return gameData{}, errors.Wrap(err, "failed to fetch second request")
	}
	defer resp.Body.Close()

	var data gameData
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return gameData{}, errors.Wrap(err, "failed to decode second request")
	}

	return data, nil
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
