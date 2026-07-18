// Command gen fetches the City of Cape Town's Official Suburb polygons from
// the open-data ArcGIS service and writes docs/data/suburbs.json: simplified
// silhouette rings (normalized to a 100x100 frame), lon/lat centroid, and
// area for each suburb. Run it manually when the source data changes; the
// output is committed so the site needs no build step.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sort"
	"strings"
	"time"

	arcgis "github.com/richardwooding/go-arcgis"
	"github.com/richardwooding/suburble/internal/geom"
)

const (
	baseURL   = "https://esapqa.capetown.gov.za/agsext/rest/services/Theme_Based/ODP_SPLIT_5/FeatureServer"
	layerID   = 3
	nameField = "OFC_SBRB_NAME"

	// Simplification tolerance is RELATIVE to each suburb's extent — an
	// absolute tolerance flattens small suburbs into blobs while barely
	// touching big ones (Wellway Park went from 63 boundary points to 6).
	simplifyRelTol = 0.008   // 0.8% of the suburb's larger span
	simplifyMinDeg = 0.00001 // ~1m floor
	frameSize      = 100.0
	maxRings       = 4    // keep at most this many rings per suburb
	minRingShare   = 0.02 // drop rings under 2% of the suburb's area
)

// Suburb is one entry in the generated dataset.
type Suburb struct {
	Name   string         `json:"name"`
	Center [2]float64     `json:"c"` // lon, lat
	AreaKm float64        `json:"km2"`
	Known  bool           `json:"known,omitempty"` // in the curated normal-mode answer pool
	Rings  [][][2]float64 `json:"rings"`           // normalized to frameSize x frameSize
}

type dataset struct {
	Generated string   `json:"generated"`
	Source    string   `json:"source"`
	Attrib    string   `json:"attribution"`
	Frame     float64  `json:"frame"`
	Suburbs   []Suburb `json:"suburbs"`
}

type geoPolygon struct {
	Type        string          `json:"type"`
	Coordinates json.RawMessage `json:"coordinates"`
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "gen:", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	client := arcgis.NewClient(baseURL, arcgis.WithTimeout(2*time.Minute))
	features, err := client.Layer(layerID).Query().
		Fields(nameField).
		Format(arcgis.FormatGeoJSON).
		All(ctx)
	if err != nil {
		return fmt.Errorf("fetch suburbs: %w", err)
	}
	fmt.Fprintf(os.Stderr, "fetched %d features\n", len(features))

	// Group rings by suburb name — the layer can hold multiple polygons per
	// suburb (enclaves, coastal islands).
	byName := map[string][][]geom.Point{}
	for _, f := range features {
		name := strings.TrimSpace(featureName(f))
		if name == "" {
			continue
		}
		rings, err := parseRings(f.Geometry)
		if err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		byName[name] = append(byName[name], rings...)
	}

	known := map[string]bool{}
	for _, n := range curated {
		known[n] = true
	}

	var suburbs []Suburb
	for name, rings := range byName {
		s, ok := buildSuburb(name, rings)
		if !ok {
			fmt.Fprintf(os.Stderr, "skipping %s: degenerate geometry\n", name)
			continue
		}
		if known[name] {
			s.Known = true
			delete(known, name)
		}
		suburbs = append(suburbs, s)
	}
	sort.Slice(suburbs, func(i, j int) bool { return suburbs[i].Name < suburbs[j].Name })

	// Every curated name must exist in the dataset — fail loudly on drift.
	if len(known) > 0 {
		var missing []string
		for n := range known {
			missing = append(missing, n)
		}
		sort.Strings(missing)
		return fmt.Errorf("curated names not in dataset: %s", strings.Join(missing, ", "))
	}

	out := dataset{
		Generated: time.Now().UTC().Format(time.RFC3339),
		Source:    fmt.Sprintf("%s/%d (%s)", baseURL, layerID, nameField),
		Attrib:    "Data: City of Cape Town Open Data Portal",
		Frame:     frameSize,
		Suburbs:   suburbs,
	}
	f, err := os.Create("docs/data/suburbs.json")
	if err != nil {
		return err
	}
	if err := json.NewEncoder(f).Encode(out); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "wrote %d suburbs\n", len(suburbs))
	return nil
}

func featureName(f arcgis.Feature) string {
	for _, m := range []map[string]any{f.Properties, f.Attributes} {
		if v, ok := m[nameField].(string); ok {
			return v
		}
	}
	return ""
}

// parseRings accepts GeoJSON Polygon or MultiPolygon and returns exterior
// rings only (holes don't read at silhouette scale).
func parseRings(raw json.RawMessage) ([][]geom.Point, error) {
	var g geoPolygon
	if err := json.Unmarshal(raw, &g); err != nil {
		return nil, err
	}
	toRing := func(coords [][]float64) []geom.Point {
		ring := make([]geom.Point, 0, len(coords))
		for _, c := range coords {
			if len(c) >= 2 {
				ring = append(ring, geom.Point{X: c[0], Y: c[1]})
			}
		}
		return ring
	}
	switch g.Type {
	case "Polygon":
		var coords [][][]float64
		if err := json.Unmarshal(g.Coordinates, &coords); err != nil {
			return nil, err
		}
		if len(coords) == 0 {
			return nil, nil
		}
		return [][]geom.Point{toRing(coords[0])}, nil
	case "MultiPolygon":
		var coords [][][][]float64
		if err := json.Unmarshal(g.Coordinates, &coords); err != nil {
			return nil, err
		}
		var rings [][]geom.Point
		for _, poly := range coords {
			if len(poly) > 0 {
				rings = append(rings, toRing(poly[0]))
			}
		}
		return rings, nil
	default:
		return nil, fmt.Errorf("unsupported geometry type %q", g.Type)
	}
}

func buildSuburb(name string, rings [][]geom.Point) (Suburb, bool) {
	// Rank rings by area, keep the biggest few, drop slivers.
	type ranked struct {
		ring []geom.Point
		area float64
	}
	var rs []ranked
	var total float64
	for _, r := range rings {
		if len(r) < 4 {
			continue
		}
		a := math.Abs(geom.RingArea(r))
		rs = append(rs, ranked{r, a})
		total += a
	}
	if len(rs) == 0 || total == 0 {
		return Suburb{}, false
	}
	sort.Slice(rs, func(i, j int) bool { return rs[i].area > rs[j].area })

	// Tolerance scales with the suburb's own bounding box.
	minX, minY := math.Inf(1), math.Inf(1)
	maxX, maxY := math.Inf(-1), math.Inf(-1)
	for _, r := range rs {
		for _, p := range r.ring {
			minX, maxX = math.Min(minX, p.X), math.Max(maxX, p.X)
			minY, maxY = math.Min(minY, p.Y), math.Max(maxY, p.Y)
		}
	}
	tol := math.Max(simplifyMinDeg, math.Max(maxX-minX, maxY-minY)*simplifyRelTol)

	var kept [][]geom.Point
	for _, r := range rs {
		if len(kept) >= maxRings || r.area/total < minRingShare {
			break
		}
		kept = append(kept, geom.Simplify(r.ring, tol))
	}

	center := geom.Centroid(rs[0].ring)
	norm := geom.Normalize(kept, frameSize, frameSize)

	out := make([][][2]float64, len(norm))
	for i, ring := range norm {
		out[i] = make([][2]float64, len(ring))
		for j, p := range ring {
			out[i][j] = [2]float64{p.X, p.Y}
		}
	}
	// Area in km²: shoelace in degrees, corrected for latitude.
	cosLat := math.Cos(center.Y * math.Pi / 180)
	kmPerDegLat := 111.32
	areaKm := total * kmPerDegLat * kmPerDegLat * cosLat

	return Suburb{
		Name:   name,
		Center: [2]float64{round4(center.X), round4(center.Y)},
		AreaKm: math.Round(areaKm*100) / 100,
		Rings:  out,
	}, true
}

func round4(v float64) float64 { return math.Round(v*1e4) / 1e4 }
