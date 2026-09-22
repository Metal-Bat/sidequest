package web

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/Metal-Bat/sidequest/internal/quest"
	"github.com/jackc/pgx/v5"
)

func (a *App) load(r *http.Request, d *Data) error {
	qs, err := a.quests.List(r.Context(), d.User.ID)
	d.Quests = qs
	return err
}

func (a *App) home(w http.ResponseWriter, r *http.Request) {
	d := a.data(r)
	d.Page = "home"
	if err := a.load(r, &d); err != nil {
		a.fail(w, r, 500, "The ledger could not be opened.", err)
		return
	}
	a.render(w, 200, "layout", d)
}

func (a *App) newQuest(w http.ResponseWriter, r *http.Request) {
	d := a.data(r)
	d.Page = "new"
	name := "layout"
	if isHX(r) {
		name = "quest-form"
	}
	a.render(w, 200, name, d)
}

func (a *App) create(w http.ResponseWriter, r *http.Request) {
	d := a.data(r)
	d.Page = "new"
	d.Title = r.PostForm.Get("title")
	d.Description = r.PostForm.Get("description")
	d.Difficulty = r.PostForm.Get("difficulty")
	d.Estimate = r.PostForm.Get("minutes")
	difficulty, _ := strconv.Atoi(d.Difficulty)
	minutes, _ := strconv.Atoi(d.Estimate)
	q := quest.Quest{UserID: d.User.ID, Title: d.Title, Description: d.Description, Difficulty: difficulty, EstimatedMinutes: minutes}
	if d.Error = q.Validate(); d.Error != "" {
		name := "layout"
		if isHX(r) {
			name = "quest-form"
		}
		a.render(w, 422, name, d)
		return
	}
	q, err := a.quests.Create(r.Context(), q)
	if err != nil {
		a.fail(w, r, 500, "The quest could not be recorded.", err)
		return
	}
	if !isHX(r) {
		redirect(w, r, "/")
		return
	}
	d.Changed = q
	d.Action = "create"
	d.Message = "A new quest has joined the board."
	a.render(w, 200, "mutation", d)
}

func (a *App) pick(w http.ResponseWriter, r *http.Request) {
	d := a.data(r)
	d.Page = "home"
	minutes, err := strconv.Atoi(r.URL.Query().Get("minutes"))
	if err != nil || (minutes != 0 && minutes != 15 && minutes != 30 && minutes != 60) {
		a.fail(w, r, 400, "Choose 15, 30, 60 minutes, or any duration.", nil)
		return
	}
	d.Minutes = minutes
	var previous int64
	if raw := r.URL.Query().Get("previous"); raw != "" {
		previous, err = strconv.ParseInt(raw, 10, 64)
		if err != nil || previous < 1 {
			a.fail(w, r, 400, "Invalid previous quest.", nil)
			return
		}
	}
	q, err := a.quests.Pick(r.Context(), d.User.ID, minutes, previous)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		a.fail(w, r, 500, "The guildmaster could not consult the ledger.", err)
		return
	}
	if err == nil {
		d.Selected = &q
	} else {
		d.Message = "No unclaimed quest fits this time. Try a longer duration or add a new quest."
	}
	name := "card"
	if !isHX(r) {
		name = "layout"
		if err = a.load(r, &d); err != nil {
			a.fail(w, r, 500, "The ledger could not be opened.", err)
			return
		}
	}
	a.render(w, 200, name, d)
}

func (a *App) mutate(w http.ResponseWriter, r *http.Request) {
	d := a.data(r)
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		a.fail(w, r, 400, "Invalid quest number.", nil)
		return
	}
	action := r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]
	q, err := a.quests.Mutate(r.Context(), d.User.ID, id, action)
	if errors.Is(err, pgx.ErrNoRows) {
		a.fail(w, r, 404, "That quest was not found in your ledger.", nil)
		return
	}
	if errors.Is(err, quest.ErrTransition) {
		a.fail(w, r, 409, "That quest has already closed. Refresh the ledger to see its current state.", nil)
		return
	}
	if err != nil {
		a.fail(w, r, 500, "The quest could not be updated.", err)
		return
	}
	if !isHX(r) {
		redirect(w, r, "/")
		return
	}
	if err = a.pool.QueryRow(r.Context(), `SELECT xp FROM users WHERE id=$1`, d.User.ID).Scan(&d.User.XP); err != nil {
		a.fail(w, r, 500, "Please refresh the ledger to see your updated experience.", err)
		return
	}
	d.Action = action
	d.Changed = q
	d.Message = fmt.Sprintf("Quest updated: %s.", q.Title)
	d.OOB = true
	a.render(w, 200, "mutation", d)
}
