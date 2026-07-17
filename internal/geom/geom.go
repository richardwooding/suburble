// Package geom holds the small geometry toolkit the generator needs:
// Douglas-Peucker simplification, polygon area/centroid, haversine distance,
// and normalization of a lon/lat ring into an SVG-friendly local frame.
package geom

import "math"

// Point is a lon/lat (or local x/y) coordinate.
type Point struct{ X, Y float64 }

// Simplify runs Douglas-Peucker on a ring with the given tolerance (in the
// ring's own units). Endpoints are always kept; the ring should be closed or
// open consistently — output matches input convention.
func Simplify(pts []Point, tol float64) []Point {
	if len(pts) <= 2 {
		return pts
	}
	keep := make([]bool, len(pts))
	keep[0], keep[len(pts)-1] = true, true
	simplifySeg(pts, 0, len(pts)-1, tol, keep)
	out := make([]Point, 0, len(pts))
	for i, k := range keep {
		if k {
			out = append(out, pts[i])
		}
	}
	return out
}

func simplifySeg(pts []Point, lo, hi int, tol float64, keep []bool) {
	if hi-lo < 2 {
		return
	}
	maxD, maxI := 0.0, -1
	for i := lo + 1; i < hi; i++ {
		if d := perpDist(pts[i], pts[lo], pts[hi]); d > maxD {
			maxD, maxI = d, i
		}
	}
	if maxD > tol {
		keep[maxI] = true
		simplifySeg(pts, lo, maxI, tol, keep)
		simplifySeg(pts, maxI, hi, tol, keep)
	}
}

func perpDist(p, a, b Point) float64 {
	dx, dy := b.X-a.X, b.Y-a.Y
	l2 := dx*dx + dy*dy
	if l2 == 0 {
		return math.Hypot(p.X-a.X, p.Y-a.Y)
	}
	t := ((p.X-a.X)*dx + (p.Y-a.Y)*dy) / l2
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	return math.Hypot(p.X-(a.X+t*dx), p.Y-(a.Y+t*dy))
}

// RingArea returns the signed shoelace area of a ring (input units squared).
func RingArea(pts []Point) float64 {
	var s float64
	for i := range pts {
		j := (i + 1) % len(pts)
		s += pts[i].X*pts[j].Y - pts[j].X*pts[i].Y
	}
	return s / 2
}

// Centroid returns the area-weighted centroid of a ring.
func Centroid(pts []Point) Point {
	var cx, cy, a float64
	for i := range pts {
		j := (i + 1) % len(pts)
		cross := pts[i].X*pts[j].Y - pts[j].X*pts[i].Y
		cx += (pts[i].X + pts[j].X) * cross
		cy += (pts[i].Y + pts[j].Y) * cross
		a += cross
	}
	if a == 0 {
		return pts[0]
	}
	return Point{cx / (3 * a), cy / (3 * a)}
}

// HaversineKm returns the great-circle distance between two lon/lat points.
func HaversineKm(a, b Point) float64 {
	const r = 6371.0
	la1, la2 := a.Y*math.Pi/180, b.Y*math.Pi/180
	dla := la2 - la1
	dlo := (b.X - a.X) * math.Pi / 180
	h := math.Sin(dla/2)*math.Sin(dla/2) + math.Cos(la1)*math.Cos(la2)*math.Sin(dlo/2)*math.Sin(dlo/2)
	return 2 * r * math.Asin(math.Sqrt(h))
}

// BearingDeg returns the initial great-circle bearing from a to b in degrees
// clockwise from north.
func BearingDeg(a, b Point) float64 {
	la1, la2 := a.Y*math.Pi/180, b.Y*math.Pi/180
	dlo := (b.X - a.X) * math.Pi / 180
	y := math.Sin(dlo) * math.Cos(la2)
	x := math.Cos(la1)*math.Sin(la2) - math.Sin(la1)*math.Cos(la2)*math.Cos(dlo)
	deg := math.Atan2(y, x) * 180 / math.Pi
	return math.Mod(deg+360, 360)
}

// Normalize maps rings into a width x height frame (preserving aspect,
// centered, y flipped for SVG) after correcting longitude for latitude
// foreshortening, and rounds coordinates to one decimal.
func Normalize(rings [][]Point, width, height float64) [][]Point {
	if len(rings) == 0 {
		return nil
	}
	minX, minY := math.Inf(1), math.Inf(1)
	maxX, maxY := math.Inf(-1), math.Inf(-1)
	// latitude correction so shapes aren't horizontally squashed
	midLat := rings[0][0].Y
	cosLat := math.Cos(midLat * math.Pi / 180)
	for _, ring := range rings {
		for _, p := range ring {
			x := p.X * cosLat
			minX, maxX = math.Min(minX, x), math.Max(maxX, x)
			minY, maxY = math.Min(minY, p.Y), math.Max(maxY, p.Y)
		}
	}
	spanX, spanY := maxX-minX, maxY-minY
	scale := math.Min(width/spanX, height/spanY)
	offX := (width - spanX*scale) / 2
	offY := (height - spanY*scale) / 2

	out := make([][]Point, len(rings))
	for i, ring := range rings {
		out[i] = make([]Point, len(ring))
		for j, p := range ring {
			x := (p.X*cosLat-minX)*scale + offX
			y := height - ((p.Y-minY)*scale + offY) // flip for SVG
			out[i][j] = Point{round1(x), round1(y)}
		}
	}
	return out
}

func round1(v float64) float64 { return math.Round(v*10) / 10 }
