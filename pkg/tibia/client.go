package tibia

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Client struct {
	httpClient *http.Client
	baseURL    string
}

func NewClient(baseURL string) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 45 * time.Second, // Allow time for multi-page scraping (10 pages * 3s each + buffer)
		},
		baseURL: baseURL,
	}
}

type Character struct {
	Name      string `json:"name"`
	Sex       string `json:"sex"`
	Vocation  string `json:"vocation"`
	Level     int    `json:"level"`
	Residence string `json:"residence"`
	Guild     string `json:"guild"`
	GuildRank string `json:"guild_rank"`
	GuildURL  string `json:"guild_url"`
	LastLogin string `json:"last_login"`
	IsPremium bool   `json:"is_premium"`
	Country   string `json:"country"`
}

type GuildMember struct {
	Rank     string `json:"rank"`
	Name     string `json:"name"`
	Vocation string `json:"vocation"`
	Level    int    `json:"level"`
	Status   string `json:"status"`
}

type GuildResponse struct {
	GuildID int           `json:"guild_id"`
	Members []GuildMember `json:"members"`
	Total   int           `json:"total"`
}

type Powergamer struct {
	Name           string `json:"name"`
	Vocation       string `json:"vocation"`
	Level          int    `json:"level"`
	ExperienceGain int64  `json:"experience_gain"`
	LevelGain      int    `json:"level_gain"`
}

func (c *Client) GetCharacter(name string) (*Character, error) {
	url := fmt.Sprintf("%s/characters/%s", c.baseURL, name)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch character: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var character Character
	if err := json.NewDecoder(resp.Body).Decode(&character); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &character, nil
}

func (c *Client) GetGuildMembers(guildID int) (*GuildResponse, error) {
	url := fmt.Sprintf("%s/guilds/%d", c.baseURL, guildID)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch guild: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var guild GuildResponse
	if err := json.NewDecoder(resp.Body).Decode(&guild); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &guild, nil
}

// GetPowergamers fetches powergamers list from the API
// list: "today", "lastday", "last2days", etc. (default: "today")
// vocation: "", "0" (no vocation), "1" (sorcerers), "2" (druids), "3" (paladins), "4" (knights) (default: "")
// includeAll: if true, fetches all pages; if false, fetches only first page (default: false)
func (c *Client) GetPowergamers(list, vocation string, includeAll bool) ([]Powergamer, error) {
	if list == "" {
		list = "today"
	}

	includeAllStr := "false"
	if includeAll {
		includeAllStr = "true"
	}

	url := fmt.Sprintf("%s/powergamers?list=%s&include_all=%s&vocation=%s", c.baseURL, list, includeAllStr, vocation)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch powergamers: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var powergamers []Powergamer
	if err := json.NewDecoder(resp.Body).Decode(&powergamers); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return powergamers, nil
}
