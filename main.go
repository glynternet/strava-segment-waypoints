package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/tkrajina/gpxgo/gpx"
)

const usage = "Usage: strava-segment-waypoints --token <token> <segment_id> [<segment_id>...]"

func main() {
	token, segmentIDs, err := setup()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
		_, _ = fmt.Fprintln(os.Stderr, usage)
		os.Exit(22)
	}
	if err := writeSegments(os.Stdout, token, segmentIDs...); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "Error: "+err.Error())
	}
}

func setup() (string, []int64, error) {
	token := flag.String("token", "", "Strava API token")
	flag.Parse()
	if *token == "" {
		return "", nil, errors.New("must provide token using --token flag")
	}

	if len(flag.Args()) == 0 {
		return "", nil, errors.New("must provide at least one segment ID")
	}
	var segmentIDs []int64
	for _, arg := range flag.Args() {
		id, err := strconv.ParseInt(arg, 10, 64)
		if err != nil {
			return "", nil, err
		}
		segmentIDs = append(segmentIDs, id)
	}
	return *token, segmentIDs, nil
}

func writeSegments(w io.Writer, token string, segmentIDs ...int64) error {
	var waypoints []gpx.GPXPoint
	for _, segmentID := range segmentIDs {
		segmentWaypoints, err := segmentWaypoints(segmentID, token)
		if err != nil {
			return fmt.Errorf("getting segment waypoints for segment:%q: %w", segmentID, err)
		}
		waypoints = append(waypoints, segmentWaypoints...)
	}
	xml, err := (&gpx.GPX{
		Creator: "https://www.github.com/glynternet/strava-segment-waypoints",
		// TODO: add flag for name
		Name:      "",
		Waypoints: waypoints,
	}).ToXml(gpx.ToXmlParams{})
	if err != nil {
		return fmt.Errorf("creating GPX XML output: %w", err)
	}
	if _, err := w.Write(xml); err != nil {
		return fmt.Errorf("writing GPX XML output: %w", err)
	}
	return nil
}

func segmentWaypoints(segmentID int64, token string) ([]gpx.GPXPoint, error) {
	url := fmt.Sprintf("https://www.strava.com/api/v3/segments/%d", segmentID)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("generating request for segment ID:%q: %w", segmentID, err)
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("making request for segment ID:%q: %w", segmentID, err)
	}
	if res.StatusCode != 200 {
		return nil, fmt.Errorf("request for segment ID:%q returned non-200 status code: %s", segmentID, res.Status)
	}
	// https://developers.strava.com/docs/reference/#api-models-LatLng
	type stravaLatLng [2]float64
	var segment struct {
		Name  string       `json:"name"`
		ID    int64        `json:"id"`
		Start stravaLatLng `json:"start_latlng"`
		End   stravaLatLng `json:"end_latlng"`
	}
	if err := json.NewDecoder(res.Body).Decode(&segment); err != nil {
		return nil, fmt.Errorf("decoding segment response for segment ID:%q: %w", segmentID, err)
	}
	// Do not include Source field (src tag) in gpx.Point below as it may cause waypoints
	// to not be able to be successfully uploaded.
	// https://github.com/glynternet/strava-segment-waypoints/issues/2
	return []gpx.GPXPoint{{
		Point: gpx.Point{
			Latitude:  segment.Start[0],
			Longitude: segment.Start[1],
			Elevation: gpx.NullableFloat64{},
		},
		Name:   segment.Name + " (start)",
		Symbol: "Flag, Green",
		Type:   "user",
	}, {
		Point: gpx.Point{
			Latitude:  segment.End[0],
			Longitude: segment.End[1],
			Elevation: gpx.NullableFloat64{},
		},
		Name:   segment.Name + " (end)",
		Symbol: "Flag, Red",
		Type:   "user",
	}}, nil
}
