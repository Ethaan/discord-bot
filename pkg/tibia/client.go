package tibia

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	httpClient   *http.Client
	baseURL      string
	proxyBaseURL string
}

func NewClient(baseURL, proxyBaseURL string) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 45 * time.Second,
		},
		baseURL:      strings.TrimRight(baseURL, "/"),
		proxyBaseURL: strings.TrimRight(proxyBaseURL, "/"),
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
	Rank     int    `json:"rank"`
	Name     string `json:"name"`
	Vocation string `json:"vocation"`
	Level    int    `json:"level"`
	Today    int    `json:"today"`
}

type PowergamersResponse struct {
	Powergamers []Powergamer `json:"power_gamers"`
	Total       int          `json:"total"`
}

type OnlinePlayer struct {
	Name     string `json:"name"`
	Level    int    `json:"level"`
	Vocation string `json:"vocation"`
	Country  string `json:"country"`
}

type WhosOnlineResponse struct {
	Players []OnlinePlayer `json:"players"`
}

type characterResponse struct {
	Character struct {
		Character struct {
			Name      string `json:"name"`
			Sex       string `json:"sex"`
			Vocation  string `json:"vocation"`
			Level     int    `json:"level"`
			Residence string `json:"residence"`
			Guild     struct {
				Name string `json:"name"`
				Rank string `json:"rank"`
			} `json:"guild"`
			LastLogin     string `json:"last_login"`
			AccountStatus string `json:"account_status"`
		} `json:"character"`
	} `json:"character"`
}

func (c *Client) GetCharacter(name string) (*Character, error) {
	endpoint := fmt.Sprintf("%s/character/%s", c.baseURL, url.PathEscape(name))

	resp, err := c.httpClient.Get(endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch character: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var response characterResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	data := response.Character.Character
	return &Character{
		Name:      data.Name,
		Sex:       data.Sex,
		Vocation:  data.Vocation,
		Level:     data.Level,
		Residence: data.Residence,
		Guild:     data.Guild.Name,
		GuildRank: data.Guild.Rank,
		LastLogin: data.LastLogin,
		IsPremium: data.AccountStatus == "Premium Account",
	}, nil
}

func (c *Client) GetGuildMembers(guildID int) (*GuildResponse, error) {
	endpoint := fmt.Sprintf("%s/guilds/%d", c.proxyBaseURL, guildID)

	resp, err := c.httpClient.Get(endpoint)
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

func (c *Client) GetPowergamers(list, vocation string, includeAll bool) ([]Powergamer, error) {
	if list == "" {
		list = "today"
	}

	includeAllStr := "false"
	if includeAll {
		includeAllStr = "true"
	}

	endpoint := fmt.Sprintf("%s/powergamers?list=%s&include_all=%s&vocation=%s", c.proxyBaseURL, list, includeAllStr, vocation)

	resp, err := c.httpClient.Get(endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch powergamers: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var response PowergamersResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return response.Powergamers, nil
}

func (c *Client) GetWhosOnline() (*WhosOnlineResponse, error) {
	endpoint := fmt.Sprintf("%s/whoisonline", c.proxyBaseURL)

	resp, err := c.httpClient.Get(endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch whos online: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var response WhosOnlineResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &response, nil
}
