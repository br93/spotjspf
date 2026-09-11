# spotjspf

A tiny, dependency-free Go web app that:

- serves a single-page upload UI,
- parses [JSPF](https://www.jspf.org/) playlists natively (no external Go libs),
- kicks off a `spotdl` download for every track **as soon as the file is uploaded**,
- shows a live-updating list of recent downloads (queued / downloading / completed / failed).

## Run it

```bash
docker compose up -d --build
```

Then open `http://localhost:8090` (or whatever `PORT` you set), and upload a `.jspf` playlist file.

## How track matching works

For each track, spotjspf resolves a query for `spotdl download` in this order:

1. **Direct Spotify link** — if the track's `identifier` or `location` already contains a Spotify URL/URI (`open.spotify.com/...` or `spotify:track:...`), that's used as-is. Most accurate, no lookup needed.
2. **MusicBrainz \u2192 Spotify** — if instead the track has a MusicBrainz recording ID (common in ListenBrainz playlist exports, e.g. `identifier: ["https://musicbrainz.org/recording/<mbid>"]`), spotjspf looks the recording up on the MusicBrainz API and checks its relationships for a linked Spotify track. If found, that exact Spotify URL is used. This lookup happens right before the download starts (not at upload time), so uploads still queue instantly.
3. **Text search** — otherwise (or if the MusicBrainz lookup finds no Spotify link), it falls back to `"<creator> - <title>"` as a text search. If only a title exists, it searches on the title alone.

The "recent downloads" list shows a **Match** column so you can see which path was used for each track (`Spotify link`, `MusicBrainz → Spotify`, or `Text search`).

MusicBrainz's API usage policy requires a descriptive User-Agent and a rate limit of about 1 request/second — spotjspf enforces both automatically (see `MB_USER_AGENT` / `MB_RATE_LIMIT_MS` below), and caches lookups per MBID for the life of the process.

## Configuration (environment variables)

| Variable            | Default      | Meaning                                                          |
|----------------------|--------------|-------------------------------------------------------------------|
| `PORT`               | `:8080`       | Host port the UI is exposed on (.env only)              |
| `HOST_DOWNLOAD_DIR`  | `./downloads`| Host folder mapped to `/downloads` inside the container           |
| `HOST_DATA_DIR`      | `./data`     | Host folder for the persisted download-history JSON               |
| `CONCURRENCY`        | `2`          | Number of simultaneous `spotdl` downloads                         |
| `SPOTDL_FORMAT`      | `mp3`        | Output audio format (`mp3`, `flac`, `opus`, `m4a`, `wav`, `ogg`)   |
| `SPOTDL_ARGS`        | *(empty)*    | Extra raw args appended to every `spotdl download` call            |
| `MAX_HISTORY`        | `200`        | How many recent jobs to keep in the list/history file             |
| `MB_LOOKUP_ENABLED`  | `true`       | Resolve MusicBrainz recording IDs to Spotify links before searching |
| `MB_USER_AGENT`      | *(placeholder)* | **Change this** — MusicBrainz asks for contact info (eg. github) |
| `MB_RATE_LIMIT_MS`   | `1000`       | Minimum ms between MusicBrainz API calls                          |

## Local dev (without Docker)

Requires Go 1.22+, plus `spotdl` and `ffmpeg` installed and on your `PATH`.

```bash
go build -o spotjspf .
DOWNLOAD_DIR=./downloads DATA_DIR=./data ./spotjspf
```

## Notes

- Uploads are capped at 10 MB (plenty for a JSPF playlist).
- Each download job runs with a 20-minute timeout.
- The JSPF parser is lenient about `identifier`/`location` being either a single string or an array, since real-world exporters aren't all spec-perfect.
