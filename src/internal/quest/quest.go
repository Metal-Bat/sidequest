package quest

import (
	"strings"
	"time"
	"unicode/utf8"
)

type Quest struct {
	ID, UserID                             int64
	Title, Description                     string
	Difficulty, EstimatedMinutes, RewardXP int
	Status                                 string
	CreatedAt                              time.Time
	CompletedAt                            *time.Time
}

func Reward(difficulty int) int {
	if difficulty < 1 || difficulty > 5 {
		return 0
	}
	return [...]int{20, 40, 70, 110, 170}[difficulty-1]
}

func Level(xp int) (level, progress int) {
	return xp/200 + 1, xp % 200
}

func (q Quest) Validate() string {
	if n := utf8.RuneCountInString(strings.TrimSpace(q.Title)); n < 1 || n > 120 {
		return "Give your quest a title of 1–120 characters."
	}
	if utf8.RuneCountInString(q.Description) > 4000 {
		return "Keep the description within 4,000 characters."
	}
	if q.Difficulty < 1 || q.Difficulty > 5 {
		return "Choose a difficulty from 1 to 5."
	}
	if q.EstimatedMinutes < 1 || q.EstimatedMinutes > 10080 {
		return "Estimate between 1 and 10,080 minutes."
	}
	return ""
}
