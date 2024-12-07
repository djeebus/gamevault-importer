package cmd

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path/filepath"
	"time"

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

	var rootPath string
	var filterGameID string

	args := os.Args
	switch len(args) {
	case 2:
		rootPath = args[1]
	case 3:
		rootPath = args[1]
		filterGameID = args[2]
	default:
		fmt.Printf("Usage: %s PATH [GAMEID]\n", args[0])
		os.Exit(1)
	}

	gogURL, err := url.Parse("https://www.gog.com")
	if err != nil {
		panic(err)
	}

	jar, err := cookiejar.New(nil)
	if err != nil {
		panic(err)
	}
	jar.SetCookies(gogURL, []*http.Cookie{
		{Name: "gog-al", Value: os.Getenv("AUTH_GOG_AL"), Domain: "gog.com"},
		{Name: "gog_lc", Value: os.Getenv("AUTH_GOG_LC"), Domain: "gog.com"},
		{Name: "gog_us", Value: os.Getenv("AUTH_GOG_US"), Domain: "gog.com"},
	})
	client.Jar = jar

	gameIDs, err := getGameIDs(client)
	if err != nil {
		return err
	}

	for _, gameID := range gameIDs {
		if filterGameID != "" && gameID != filterGameID {
			continue
		}

		gameData, err := getGameData(gameID, &client)
		if err != nil {
			return errors.Wrapf(err, "failed to get game data for %s", gameID)
		}

		files, err := getDownloads(gameData)
		if err != nil {
			return errors.Wrapf(err, "failed to get game data for %s", gameID)
		}

		filename := createFilename(gameData, files[0])
		filename = filepath.Join(rootPath, filename)

		if _, err := os.Stat(filename); err == nil {
			fmt.Printf("%q already downloaded, skipping\n", filename)
		}

		fmt.Printf("Found %d files, downloading\n", len(files))
		if err = createGamePackage(filename, files, &client); err != nil {
			return errors.Wrapf(err, "failed to create game package for %s", gameID)
		}
	}

	return nil
}

func createGamePackage(zipFilename string, files []file, client *http.Client) error {
	file, err := os.Create(zipFilename)
	if err != nil {
		return errors.Wrap(err, "failed to create file")
	}
	defer file.Close()

	zipWriter := zip.NewWriter(file)
	defer zipWriter.Close()

	for _, f := range files {
		fileUrl, err := url.JoinPath("https://www.gog.com", f.ManualURL)
		if err != nil {
			return errors.Wrap(err, "failed to make file url")
		}

		req, err := http.NewRequest("GET", fileUrl, nil)
		if err != nil {
			return errors.Wrap(err, "failed to create request")
		}

		res, err := client.Do(req)
		if err != nil {
			return errors.Wrap(err, "failed to download file")
		}
		defer res.Body.Close()

		_, filename := filepath.Split(res.Request.URL.Path)

		w, err := zipWriter.Create(filename)
		if err != nil {
			return errors.Wrap(err, "failed to create zip file")
		}

		if _, err = io.Copy(w, res.Body); err != nil {
			return errors.Wrap(err, "failed to copy file")
		}
	}

	return nil
}

func createFilename(data gameData, file file) string {
	release := time.Unix(data.ReleaseTimestamp, 0)

	return fmt.Sprintf("%s (%s) (%d).zip", data.Title, file.Version, release.Year())
}

func getGameData(gameID string, client *http.Client) (gameData, error) {
	var data gameData

	// GET https://www.gog.com/account/gameDetails/$id.json
	url := fmt.Sprintf("https://www.gog.com/account/gameDetails/%s.json", gameID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return data, errors.Wrap(err, "failed to create second request")
	}

	resp, err := client.Do(req)
	if err != nil {
		return data, errors.Wrap(err, "failed to fetch second request")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return data, errors.Wrap(err, "failed to read second request")
	}

	if err = os.WriteFile(fmt.Sprintf("%s.json", gameID), body, 0644); err != nil {
		return data, errors.Wrap(err, "failed to write second request")
	}

	if err := json.Unmarshal(body, &data); err != nil {
		return data, errors.Wrap(err, "failed to decode second request")
	}

	return data, nil
}

func getDownloads(data gameData) ([]file, error) {
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

func getGameIDs(client http.Client) ([]string, error) {
	req, err := http.NewRequest("GET", "https://menu.gog.com/v1/account/licences", nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create first request")
	}

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
