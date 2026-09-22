package quest

import "testing"

func TestRewardsAndLevels(t *testing.T) {
	for difficulty, want := range []int{0, 20, 40, 70, 110, 170, 0} {
		if got := Reward(difficulty); got != want {
			t.Errorf("Reward(%d)=%d want %d", difficulty, got, want)
		}
	}
	for _, tt := range []struct{ xp, level, progress int }{{0, 1, 0}, {199, 1, 199}, {200, 2, 0}, {620, 4, 20}, {10000, 51, 0}} {
		l, p := Level(tt.xp)
		if l != tt.level || p != tt.progress {
			t.Errorf("Level(%d)=%d,%d", tt.xp, l, p)
		}
	}
}

func TestValidation(t *testing.T) {
	valid := Quest{Title: "A small adventure", Difficulty: 1, EstimatedMinutes: 15}
	if valid.Validate() != "" {
		t.Fatal(valid.Validate())
	}
	for _, change := range []func(*Quest){func(q *Quest) { q.Title = "  " }, func(q *Quest) { q.Difficulty = 0 }, func(q *Quest) { q.Difficulty = 6 }, func(q *Quest) { q.EstimatedMinutes = 0 }, func(q *Quest) { q.EstimatedMinutes = 10081 }, func(q *Quest) {
		for range 4001 {
			q.Description += "a"
		}
	}} {
		q := valid
		change(&q)
		if q.Validate() == "" {
			t.Fatalf("accepted invalid quest: %+v", q)
		}
	}
}
