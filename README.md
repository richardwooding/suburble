# Cape Town Suburble

The daily Cape Town suburb puzzle. Guess which of the city's **778 official
suburbs** matches the silhouette — six guesses, and every miss tells you how
far off you are, which direction to look, and how warm you're getting.

Part of the -dle family tree: [Wordle](https://www.nytimes.com/games/wordle/)
invented the daily share-grid, [Worldle](https://worldle.teuteuf.fr/) brought
it to map silhouettes, and [Canberra Suburble](https://www.suburble.au/) plays
the same suburb-guessing game for Australia's capital (and got to the name
first — this is the Cape Town edition, built independently and credited
gladly).

**▶ Play: https://richardwooding.github.io/suburble/**

A new puzzle appears at midnight SAST. Solve it, share your emoji grid,
defend your streak.

**Normal mode** draws answers from a curated pool of 113 well-known suburbs
(the full 778 stay guessable); **hard mode** draws from everything — the
official layer subdivides famously (BELHAR EXT 1–14, DELFT 3…), so hard
mode is genuinely brutal. Hints unlock after the 3rd miss (area) and 4th
miss (first letter). The curated list lives in `cmd/gen/curated.go` and is
verified against the dataset at generation time.

## How it works

- `cmd/gen` fetches the *Official Planning Suburbs* layer from the
  [City of Cape Town Open Data Portal](https://odp-cctegis.opendata.arcgis.com/)
  via [go-arcgis](https://github.com/richardwooding/go-arcgis), simplifies
  each polygon (Douglas-Peucker, ~40 m tolerance), and writes
  `docs/data/suburbs.json` — silhouette rings normalized to a 100×100 frame,
  plus centroid and area. The JSON is committed, so the site is fully static.
- The game is a single gloam-styled page with vanilla JS: the daily answer
  comes from a date-indexed, seeded shuffle (every visitor gets the same
  suburb), distance/bearing are haversine between centroids, and state and
  streaks live in localStorage.

## Development

```sh
go test ./...          # geometry: simplify, centroid, haversine, bearing
go run ./cmd/gen       # refresh docs/data/suburbs.json from the source layer
python3 -m http.server -d docs   # play locally
```

Data: City of Cape Town Open Data Portal — Official Planning Suburbs.
