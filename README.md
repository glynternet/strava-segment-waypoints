# strava-segment-waypoints

Generate a GPX file containing waypoints for start and end of strava segment.
The output can be saved to a file and used directly with navigation devices or applications (e.g. Garmin BaseCamp) to import waypoints for the start and end of strava segments.

## Usage
Build using `go` or run directly using `go run`
```
strava-segment-waypoints --token <token> <segment_id> [<segment_id>...]
```
