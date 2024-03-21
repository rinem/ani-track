package api

import (
	"encoding/json"
	"fmt"
	"net/http"
)

const (
	apiBaseURL = "https://api.myanimelist.net/v2/"
)

type AnimeSearchResult struct {
	Data []struct {
		Node struct {
			Title string `json:"title"`
		} `json:"node"`
	} `json:"data"`
}

func SearchAnime(query, limit, accessToken string) (*AnimeSearchResult, error) {
	searchURL := fmt.Sprintf("%sanime?q=%s&limit=%s", apiBaseURL, query, limit)
	req, err := http.NewRequest("GET", searchURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result AnimeSearchResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

type UserAnimeListResult struct {
	Data []struct {
		Node struct {
			Title string `json:"title"`
		} `json:"node"`
	} `json:"data"`
}

func GetUserAnimeList(username, limit, accessToken string) (*UserAnimeListResult, error) {
	userListURL := fmt.Sprintf("%susers/%s/animelist?fields=list_status&limit=%s", apiBaseURL, username, limit)
	req, err := http.NewRequest("GET", userListURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result UserAnimeListResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
