package slack

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/svenschwermer/gcal-notify/config"
)

type WorkingLocation int

const (
	WorkingLocationHome WorkingLocation = iota
	WorkingLocationOffice
)

type Client struct {
	token string
}

func NewClient() (*Client, error) {
	token, err := os.ReadFile(config.Cfg.SlackTokenFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read slack token file: %w", err)
	}
	token = bytes.TrimSpace(token)

	c := &Client{
		token: string(token),
	}

	return c, nil
}

func (c *Client) SetWorkingLocation(ctx context.Context, loc WorkingLocation) error {
	var body struct {
		Profile struct {
			Text       string `json:"status_text"`
			Emoji      string `json:"status_emoji"`
			Expiration int64  `json:"status_expiration"`
		} `json:"profile"`
	}

	// Expire at the end of the day
	body.Profile.Expiration = time.Now().AddDate(0, 0, 1).Truncate(24 * time.Hour).Unix()

	switch loc {
	case WorkingLocationHome:
		body.Profile.Text = "Working from home"
		body.Profile.Emoji = ":house_with_garden:"
	case WorkingLocationOffice:
		body.Profile.Text = "Working from the office"
		body.Profile.Emoji = ":office:"
	default:
		return fmt.Errorf("unexpected working location: %v", loc)
	}

	bodyBuf := new(bytes.Buffer)
	if err := json.NewEncoder(bodyBuf).Encode(body); err != nil {
		return fmt.Errorf("failed to encode request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, "https://slack.com/api/users.profile.set", bodyBuf)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("request failed with status %s", resp.Status)
	}

	var respBody struct {
		OK      bool   `json:"ok"`
		Error   string `json:"error"`
		Warning string `json:"warning"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if !respBody.OK {
		return fmt.Errorf("request failed: %s", respBody.Error)
	}
	if respBody.Warning != "" {
		log.Printf("Slack warning: %s", respBody.Warning)
	}

	return nil
}
