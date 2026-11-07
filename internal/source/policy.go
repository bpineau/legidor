package source

import "time"

const (
	currentTTL    = 30 * 24 * time.Hour
	historicalTTL = 90 * 24 * time.Hour
)

func (c *Client) cacheAge(defaultAge time.Duration) time.Duration {
	if c.MaxAge > 0 {
		return c.MaxAge
	}
	return defaultAge
}

func (c *Client) today() string {
	if c.now != nil {
		return c.now().Format("2006-01-02")
	}
	return time.Now().Format("2006-01-02")
}

func (c *Client) datedTTL(date string) time.Duration {
	if date != "" && date < c.today() {
		return historicalTTL
	}
	return currentTTL
}
