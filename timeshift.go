package radiko

import (
	"context"
	"time"

	"github.com/yyoshiki41/go-radiko/internal"
)

// TimeshiftPlaylistM3U8 returns uri.
func (c *Client) TimeshiftPlaylistM3U8(ctx context.Context, stationID string, start time.Time) (string, error) {
	prog, err := c.GetProgramByStartTime(ctx, stationID, start)
	if err != nil {
		return "", err
	}

	apiEndpoint := apiPath(apiV2, "ts/playlist.m3u8")
	req, err := c.newRequest(ctx, "POST", apiEndpoint, &Params{
		query: map[string]string{
			"station_id": stationID,
			"ft":         prog.Ft,
			"to":         prog.To,
			"l":          "15", // must?
		},
	})

	resp, err := c.CallWithAuthTokenHeader(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	return internal.GetURIFromM3U8(resp.Body)
}
