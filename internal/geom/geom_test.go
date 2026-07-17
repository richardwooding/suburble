package geom

import (
	"math"
	"testing"
)

func TestSimplifyDropsCollinear(t *testing.T) {
	line := []Point{{0, 0}, {1, 0.001}, {2, 0}, {3, -0.001}, {4, 0}}
	got := Simplify(line, 0.01)
	if len(got) != 2 {
		t.Fatalf("got %d points, want 2 (endpoints only): %v", len(got), got)
	}
}

func TestSimplifyKeepsCorners(t *testing.T) {
	square := []Point{{0, 0}, {5, 0}, {10, 0}, {10, 5}, {10, 10}, {5, 10}, {0, 10}, {0, 5}, {0, 0}}
	got := Simplify(square, 0.5)
	if len(got) != 5 {
		t.Fatalf("got %d points, want 5 (four corners + closing): %v", len(got), got)
	}
}

func TestCentroidOfSquare(t *testing.T) {
	sq := []Point{{0, 0}, {2, 0}, {2, 2}, {0, 2}}
	c := Centroid(sq)
	if math.Abs(c.X-1) > 1e-9 || math.Abs(c.Y-1) > 1e-9 {
		t.Fatalf("centroid = %v, want (1,1)", c)
	}
}

func TestHaversineCapeTown(t *testing.T) {
	// City centre to Simon's Town is roughly 30km as the crow flies.
	cbd := Point{18.4241, -33.9249}
	simons := Point{18.4392, -34.1927}
	d := HaversineKm(cbd, simons)
	if d < 28 || d < 0 || d > 32 {
		t.Fatalf("CBD to Simon's Town = %.1f km, want ~30", d)
	}
}

func TestBearing(t *testing.T) {
	cases := []struct {
		from, to Point
		want     float64
	}{
		{Point{18.4, -34.0}, Point{18.4, -33.5}, 0},   // due north
		{Point{18.4, -34.0}, Point{18.9, -34.0}, 90},  // due east (approx)
		{Point{18.4, -33.5}, Point{18.4, -34.0}, 180}, // due south
	}
	for _, c := range cases {
		got := BearingDeg(c.from, c.to)
		diff := math.Abs(got - c.want)
		if diff > 1.5 {
			t.Errorf("bearing %v -> %v = %.1f, want ~%.1f", c.from, c.to, got, c.want)
		}
	}
}

func TestNormalizeFitsFrame(t *testing.T) {
	ring := []Point{{18.40, -34.00}, {18.50, -34.00}, {18.50, -33.90}, {18.40, -33.90}}
	out := Normalize([][]Point{ring}, 100, 100)
	for _, p := range out[0] {
		if p.X < 0 || p.X > 100 || p.Y < 0 || p.Y > 100 {
			t.Fatalf("point %v outside 100x100 frame", p)
		}
	}
}
