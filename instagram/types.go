package instagram

import "time"

type hashtagSearchResponse struct {
	Data []struct {
		ID string `json:"id"`
	} `json:"data"`
}

type recentMediaResponse struct {
	Data   []Post `json:"data"`
	Paging *struct {
		Cursors *struct {
			After string `json:"after"`
		} `json:"cursors"`
		Next string `json:"next"`
	} `json:"paging"`
}

// Post is a single Instagram media object returned by the recent_media edge.
type Post struct {
	ID        string    `json:"id"`
	Caption   string    `json:"caption"`
	MediaType string    `json:"media_type"`
	MediaURL  string    `json:"media_url"`
	Permalink string    `json:"permalink"`
	Timestamp time.Time `json:"timestamp"`
}

type apiError struct {
	Error struct {
		Message string `json:"message"`
		Code    int    `json:"code"`
	} `json:"error"`
}
