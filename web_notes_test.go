package main

import (
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/notjoji/web-notes/internal/app"
	"github.com/notjoji/web-notes/internal/repository"
	"github.com/stretchr/testify/assert"
)

func TestNoteDTOMapping(t *testing.T) {
	desc := "desc"
	now := time.Now()
	yesterday := now.AddDate(0, 0, -1)
	testCases := []struct {
		name   string
		dbNote *repository.Note
		want   *app.NoteDTO
	}{
		{
			name: "completed note no deadline",
			dbNote: &repository.Note{
				ID:          1,
				UserID:      1,
				Name:        "note 1",
				Description: &desc,
				IsCompleted: true,
				CreatedAt:   pgtype.Date{Time: now, InfinityModifier: 0, Valid: true},
				DeadlineAt:  pgtype.Date{Time: now, InfinityModifier: 0, Valid: false},
			},
			want: &app.NoteDTO{
				ID:             1,
				UserID:         1,
				Name:           "note 1",
				Description:    desc,
				CreatedAt:      now.Format("2006-01-02"),
				Type:           "Завершено",
				TypeClass:      "text-white bg-success",
				StatusChangeTo: "Вернуть в работу",
			},
		},
		{
			name: "not completed yet note has deadline",
			dbNote: &repository.Note{
				ID:          2,
				UserID:      1,
				Name:        "note 2",
				Description: &desc,
				IsCompleted: false,
				CreatedAt:   pgtype.Date{Time: now, InfinityModifier: 0, Valid: true},
				DeadlineAt:  pgtype.Date{Time: now.Add(time.Hour * 24), InfinityModifier: 0, Valid: true},
			},
			want: &app.NoteDTO{
				ID:             2,
				UserID:         1,
				Name:           "note 2",
				Description:    desc,
				CreatedAt:      now.Format("2006-01-02"),
				Type:           "В работе",
				TypeClass:      "text-white bg-primary",
				StatusChangeTo: "Завершить",
			},
		},
		{
			name: "not completed yet and expired note",
			dbNote: &repository.Note{
				ID:          3,
				UserID:      1,
				Name:        "note 3",
				Description: &desc,
				IsCompleted: false,
				CreatedAt:   pgtype.Date{Time: now, InfinityModifier: 0, Valid: true},
				DeadlineAt:  pgtype.Date{Time: yesterday, InfinityModifier: 0, Valid: true},
			},
			want: &app.NoteDTO{
				ID:             3,
				UserID:         1,
				Name:           "note 3",
				Description:    desc,
				CreatedAt:      now.Format("2006-01-02"),
				Type:           "Просрочено",
				TypeClass:      "text-white bg-danger",
				StatusChangeTo: "Завершить",
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			expected := app.MapNote(testCase.dbNote)
			assert.Equal(t, expected, testCase.want)
		})
	}
}

func TestUpdateDTOMapping(t *testing.T) {
	desc := "desc"
	now := time.Now()
	yesterday := now.AddDate(0, 0, -1)
	testCases := []struct {
		name   string
		dbNote *repository.Note
		want   *app.NoteUpdateDTO
	}{
		{
			name: "completed note no deadline",
			dbNote: &repository.Note{
				ID:          1,
				UserID:      1,
				Name:        "note 1",
				Description: &desc,
				IsCompleted: true,
				CreatedAt:   pgtype.Date{Time: now, InfinityModifier: 0, Valid: true},
				DeadlineAt:  pgtype.Date{Time: now, InfinityModifier: 0, Valid: false},
			},
			want: &app.NoteUpdateDTO{
				ID:          1,
				Name:        "note 1",
				Description: desc,
				IsCompleted: true,
				HasDeadline: false,
				Deadline:    "",
			},
		},
		{
			name: "not completed yet note has deadline",
			dbNote: &repository.Note{
				ID:          2,
				UserID:      1,
				Name:        "note 2",
				Description: &desc,
				IsCompleted: false,
				CreatedAt:   pgtype.Date{Time: now, InfinityModifier: 0, Valid: true},
				DeadlineAt:  pgtype.Date{Time: now.Add(time.Hour * 24), InfinityModifier: 0, Valid: true},
			},
			want: &app.NoteUpdateDTO{
				ID:          2,
				Name:        "note 2",
				Description: desc,
				IsCompleted: false,
				HasDeadline: true,
				Deadline:    now.Add(time.Hour * 24).Format("2006-01-02"),
			},
		},
		{
			name: "not completed yet and expired note",
			dbNote: &repository.Note{
				ID:          3,
				UserID:      1,
				Name:        "note 3",
				Description: &desc,
				IsCompleted: false,
				CreatedAt:   pgtype.Date{Time: now, InfinityModifier: 0, Valid: true},
				DeadlineAt:  pgtype.Date{Time: yesterday, InfinityModifier: 0, Valid: true},
			},
			want: &app.NoteUpdateDTO{
				ID:          3,
				Name:        "note 3",
				Description: desc,
				IsCompleted: false,
				HasDeadline: true,
				Deadline:    yesterday.Format("2006-01-02"),
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			expected := app.MapNoteUpdate(testCase.dbNote)
			assert.Equal(t, expected, testCase.want)
		})
	}
}
