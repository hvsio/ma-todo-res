package instagram

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const mediaFields = "id,caption,media_type,media_url,permalink,timestamp"

// Client talks to the Instagram Graph API.
type Client struct {
	accessToken string
	userID      string
	base        string
	http        *http.Client
}

func NewClient(base, accessToken, userID string) *Client {
	return &Client{
		accessToken: accessToken,
		userID:      userID,
		base:        base,
		http:        &http.Client{Timeout: 15 * time.Second},
	}
}

// HashtagID resolves a hashtag name to its stable Graph API node ID.
// The ID is consistent per hashtag and can be cached for the session lifetime.
func (c *Client) HashtagID(hashtag string) (string, error) {
	u := fmt.Sprintf("%s/ig_hashtag_search", c.base)
	params := url.Values{
		"user_id":      {c.userID},
		"q":            {hashtag},
		"access_token": {c.accessToken},
	}

	resp, err := c.get(u + "?" + params.Encode())
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading hashtag search response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", parseAPIError(body, resp.StatusCode)
	}

	var result hashtagSearchResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("decoding hashtag search response: %w", err)
	}
	if len(result.Data) == 0 {
		return "", fmt.Errorf("hashtag %q not found", hashtag)
	}

	return result.Data[0].ID, nil
}

// RecentMedia returns posts tagged with the given hashtag node ID, ordered
// newest-first. The Graph API does not guarantee strict ordering, so callers
// should filter by timestamp themselves.
//
// TODO: implement cursor-based pagination to retrieve more than one page when
//
//	a large burst of new posts arrives between two polling ticks.
func (c *Client) RecentMedia(hashtagID string) ([]Post, error) {
	u := fmt.Sprintf("%s/%s/recent_media", c.base, hashtagID)
	params := url.Values{
		"user_id":      {c.userID},
		"fields":       {mediaFields},
		"access_token": {c.accessToken},
	}

	resp, err := c.get(u + "?" + params.Encode())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading recent_media response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, parseAPIError(body, resp.StatusCode)
	}

	var result recentMediaResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("decoding recent_media response: %w", err)
	}

	return result.Data, nil
}

func (c *Client) get(rawURL string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	return c.http.Do(req)
}

func parseAPIError(body []byte, status int) error {
	var apiErr apiError
	if err := json.Unmarshal(body, &apiErr); err == nil && apiErr.Error.Message != "" {
		return fmt.Errorf("instagram API error %d: %s", apiErr.Error.Code, apiErr.Error.Message)
	}
	return fmt.Errorf("unexpected HTTP %d", status)
}
