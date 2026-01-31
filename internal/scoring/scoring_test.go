package scoring

import "testing"

// DynamicPoints(initial, minimum, solveCount, decay)

func TestDynamicPointsMinimumGreaterThanInitial(t *testing.T) {
	points := DynamicPoints(100, 150, 10, 5)
	if points != 100 {
		t.Fatalf("expected 100, got %d", points)
	}
}

func TestDynamicPointsExample(t *testing.T) {
	points := DynamicPoints(500, 100, 30, 30)
	if points != 100 {
		t.Fatalf("expected 100, got %d", points)
	}
}

func TestDynamicPointsZeroSolves(t *testing.T) {
	points := DynamicPoints(250, 50, 0, 10)
	if points != 250 {
		t.Fatalf("expected 250, got %d", points)
	}
}

func TestDynamicPointsZeroDecay(t *testing.T) {
	points := DynamicPoints(200, 50, 5, 0)
	if points != 200 {
		t.Fatalf("expected 200, got %d", points)
	}
}

func TestDynamicPointsBelowMinimum(t *testing.T) {
	points := DynamicPoints(300, 100, 50, 10)
	if points != 100 {
		t.Fatalf("expected 100, got %d", points)
	}
}
