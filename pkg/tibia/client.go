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
			Timeout: 10 * time.Second,
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
