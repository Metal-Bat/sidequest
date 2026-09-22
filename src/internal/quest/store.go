package quest

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrTransition = errors.New("quest cannot make this transition")

const columns = `id,user_id,title,description,difficulty,estimated_minutes,reward_xp,status,created_at,completed_at`

type Store struct{ Pool *pgxpool.Pool }

func scan(row pgx.Row) (q Quest, err error) {
	err = row.Scan(&q.ID, &q.UserID, &q.Title, &q.Description, &q.Difficulty, &q.EstimatedMinutes, &q.RewardXP, &q.Status, &q.CreatedAt, &q.CompletedAt)
	return
}

func (s Store) Create(ctx context.Context, q Quest) (Quest, error) {
	if msg := q.Validate(); msg != "" {
		return Quest{}, errors.New(msg)
	}
	return scan(s.Pool.QueryRow(ctx, `INSERT INTO quests(user_id,title,description,difficulty,estimated_minutes,reward_xp) VALUES($1,$2,$3,$4,$5,$6) RETURNING `+columns, q.UserID, strings.TrimSpace(q.Title), q.Description, q.Difficulty, q.EstimatedMinutes, Reward(q.Difficulty)))
}

func (s Store) List(ctx context.Context, userID int64) ([]Quest, error) {
	rows, err := s.Pool.Query(ctx, `SELECT `+columns+` FROM quests WHERE user_id=$1 ORDER BY created_at DESC,id DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []Quest{}
	for rows.Next() {
		q, e := scan(rows)
		if e != nil {
			return nil, e
		}
		result = append(result, q)
	}
	return result, rows.Err()
}

func (s Store) Pick(ctx context.Context, userID int64, minutes int, previous int64) (Quest, error) {
	return scan(s.Pool.QueryRow(ctx, `SELECT `+columns+` FROM quests WHERE user_id=$1 AND status='available' AND ($2::integer=0 OR estimated_minutes<=$2) ORDER BY (id=$3),random() LIMIT 1`, userID, minutes, previous))
}

func (s Store) Mutate(ctx context.Context, userID, id int64, action string) (Quest, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return Quest{}, err
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	q, err := scan(tx.QueryRow(ctx, `SELECT `+columns+` FROM quests WHERE id=$1 AND user_id=$2 FOR UPDATE`, id, userID))
	if err != nil {
		return q, err
	}
	switch action {
	case "delete":
		_, err = tx.Exec(ctx, `DELETE FROM quests WHERE id=$1 AND user_id=$2`, id, userID)
	case "accept", "complete", "abandon":
		next := map[string]string{"accept": "active", "complete": "completed", "abandon": "abandoned"}[action]
		if q.Status == next {
			return q, tx.Commit(ctx)
		}
		if q.Status != "available" && (q.Status != "active" || action == "accept") {
			return q, ErrTransition
		}
		previous := q.Status
		q, err = scan(tx.QueryRow(ctx, `UPDATE quests SET status=$3,completed_at=CASE WHEN $3='completed' THEN NOW() ELSE NULL END WHERE id=$1 AND user_id=$2 AND status=$4 RETURNING `+columns, id, userID, next, previous))
		if err == nil && action == "complete" {
			_, err = tx.Exec(ctx, `UPDATE users SET xp=xp+$2 WHERE id=$1`, userID, q.RewardXP)
		}
	default:
		return q, ErrTransition
	}
	if err != nil {
		return q, err
	}
	return q, tx.Commit(ctx)
}
