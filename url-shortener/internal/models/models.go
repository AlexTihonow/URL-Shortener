package models

import "time"

type Link struct {
	ID          int64      `json:"-"`
	ShortCode   string     `json:"short_code"`
	OriginalURL string     `json:"original_url"`
	CreatedAt   time.Time  `json:"created_at"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
}

type ClickEvent struct {
	ShortCode string    `json:"short_code"`
	ClickedAt time.Time `json:"clicked_at"`
	UserAgent string    `json:"user_agent"`
	Referer   string    `json:"referer"`
}

type Stats struct {
	ShortCode   string     `json:"short_code"`
	OriginalURL string     `json:"original_url"`
	TotalClicks int64      `json:"total_clicks"`
	LastClick   *time.Time `json:"last_click,omitempty"`
	Last7Days   []DayCount `json:"last_7_days"`
}

type DayCount struct {
	Day    string `json:"day"`
	Clicks int64  `json:"clicks"`
}
