package app

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/julienschmidt/httprouter"
	"github.com/notjoji/web-notes/internal/repository"
	"github.com/notjoji/web-notes/internal/utils"
)

var Token = "token"

type App struct {
	ctx   context.Context
	db    *repository.Queries
	cache map[string]*repository.User
}

type PageData struct {
	Message string
}

type NoteType string

const (
	Active    NoteType = "В работе"
	Completed NoteType = "Завершено"
	Expired   NoteType = "Просрочено"
)

type NoteTypeClass string

const (
	Default NoteTypeClass = "text-white bg-primary"
	Success NoteTypeClass = "text-white bg-success"
	Danger  NoteTypeClass = "text-white bg-danger"
)

type StatusChangeTo string

const (
	ToActive    StatusChangeTo = "Вернуть в работу"
	ToCompleted StatusChangeTo = "Завершить"
)

type NoteDTO struct {
	ID             int64          `json:"id"`
	UserID         int64          `json:"userId"`
	Name           string         `json:"name"`
	Description    *string        `json:"description"`
	CreatedAt      string         `json:"createdAt"`
	Type           NoteType       `json:"type"`
	TypeClass      NoteTypeClass  `json:"typeClass"`
	StatusChangeTo StatusChangeTo `json:"statusChangeTo"`
}

type NoteUpdateDTO struct {
	ID          int64   `json:"id"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
	HasDeadline bool    `json:"hasDeadline"`
	Deadline    string  `json:"deadline"`
	IsCompleted bool    `json:"isCompleted"`
}

type NoteCreateDTO struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Deadline    string `json:"deadline"`
}

func MapNoteUpdate(note *repository.Note) *NoteUpdateDTO {
	deadline := ""
	if note.DeadlineAt.Valid {
		deadline = note.DeadlineAt.Time.Format(layoutISO)
	}
	return &NoteUpdateDTO{
		ID:          note.ID,
		Name:        note.Name,
		Description: note.Description,
		HasDeadline: note.DeadlineAt.Valid,
		Deadline:    deadline,
		IsCompleted: note.IsCompleted,
	}
}

const (
	layoutISO = "2006-01-02"
)

func MapNote(note *repository.Note) *NoteDTO {
	var noteType NoteType
	var noteTypeClass NoteTypeClass
	var statusChangeTo StatusChangeTo
	switch note.IsCompleted {
	case true:
		noteType = Completed
		noteTypeClass = Success
		statusChangeTo = ToActive
	case false:
		switch note.DeadlineAt.Valid && time.Now().After(note.DeadlineAt.Time.Add(24*time.Hour)) {
		case true:
			noteType = Expired
			noteTypeClass = Danger
			statusChangeTo = ToCompleted
		case false:
			noteType = Active
			noteTypeClass = Default
			statusChangeTo = ToCompleted
		}
	}
	return &NoteDTO{
		ID:             note.ID,
		UserID:         note.UserID,
		Name:           note.Name,
		Description:    note.Description,
		CreatedAt:      note.CreatedAt.Time.Format(layoutISO),
		Type:           noteType,
		TypeClass:      noteTypeClass,
		StatusChangeTo: statusChangeTo,
	}
}

func (a App) Routes(r *httprouter.Router) {
	r.ServeFiles("/public/*filepath", http.Dir("public"))
	r.GET("/", a.AuthNeeded(a.ShowMainPage))
	r.GET("/login", func(rw http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
		a.ShowLoginPage(rw, "")
	})
	r.POST("/login", a.Login)
	r.GET("/logout", a.Logout)
	r.GET("/register", func(rw http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
		a.ShowRegisterPage(rw, "")
	})
	r.POST("/register", a.Register)
	r.GET("/notes", a.AuthNeeded(a.ShowCreateNotePage))
	r.POST("/notes", a.AuthNeeded(a.CreateNewNote))
	r.GET("/notes/:page", a.AuthNeeded(a.ShowUpdateNotePage))
	r.POST("/search", a.AuthNeeded(a.FilterNotes))
	r.POST("/update", a.AuthNeeded(a.UpdateNote))
	r.POST("/delete/:id", a.AuthNeeded(a.DeleteNote))
	r.POST("/changeStatus", a.AuthNeeded(a.ChangeStatusNote))
}

func ParseTemplateFiles(rw http.ResponseWriter, html string) *template.Template {
	filePath := filepath.Join("public", "html", html)

	tmpl, err := template.ParseFiles(filePath)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return nil
	}
	return tmpl
}

func (a App) ShowLoginPage(rw http.ResponseWriter, message string) {
	tmpl := ParseTemplateFiles(rw, "login.html")
	data := PageData{message}
	err := tmpl.ExecuteTemplate(rw, "login", data)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}
}

func (a App) Login(rw http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	login := r.FormValue("login")
	password := r.FormValue("password")

	if login == "" || password == "" {
		a.ShowLoginPage(rw, "Необходимо указать логин и пароль!")
		return
	}
	user, err := a.db.GetUserByLoginAndPassword(a.ctx, repository.GetUserByLoginAndPasswordParams{
		Login:    login,
		Password: utils.GetHashedString(password),
	})
	if err != nil {
		a.ShowLoginPage(rw, "Вы ввели неверный логин или пароль!")
		return
	}

	token := utils.GetAuthToken(user)
	a.cache[token] = user
	ttl := 60 * time.Minute
	expiration := time.Now().Add(ttl)
	cookie := http.Cookie{Name: Token, Value: url.QueryEscape(token), Expires: expiration}
	http.SetCookie(rw, &cookie)
	http.Redirect(rw, r, "/", http.StatusSeeOther)
}

func (a App) Logout(rw http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	cookie, err := r.Cookie(Token)
	if err != nil {
		return
	}
	cookie.MaxAge = -1
	http.SetCookie(rw, cookie)
	http.Redirect(rw, r, "/login", http.StatusSeeOther)
}

func (a App) ShowRegisterPage(rw http.ResponseWriter, message string) {
	tmpl := ParseTemplateFiles(rw, "register.html")
	data := PageData{message}
	err := tmpl.ExecuteTemplate(rw, "register", data)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}
}

func (a App) FilterNotes(rw http.ResponseWriter, r *http.Request, p httprouter.Params) {
	search := strings.TrimSpace(r.FormValue("search"))

	if search == "" {
		p = append(p, httprouter.Param{Key: "message", Value: "Для поиска заметки требуется ввести значение!"})
		a.ShowMainPage(rw, r, p)
		return
	}

	p = append(p, httprouter.Param{Key: "search", Value: search})
	a.ShowMainPage(rw, r, p)
}

func (a App) ShowMainPage(rw http.ResponseWriter, _ *http.Request, p httprouter.Params) {
	var userID int64
	var err error

	userIDParam := p.ByName("userID")
	if userIDParam == "" {
		http.Error(rw, "требуется параметр 'userID'", http.StatusBadRequest)
		return
	}
	userID, err = strconv.ParseInt(userIDParam, 10, 64)
	if err != nil {
		http.Error(rw, "параметр 'userID' невалидный", http.StatusBadRequest)
		return
	}

	var notes []*repository.Note
	search := p.ByName("search")
	if search != "" {
		notes, err = a.db.GetNotesByUserIdAndSearch(a.ctx, repository.GetNotesByUserIdAndSearchParams{
			UserID:  userID,
			Column2: &search,
		})
	} else {
		notes, err = a.db.GetNotesByUserId(a.ctx, userID)
	}

	if err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}

	tmpl := ParseTemplateFiles(rw, "main.html")
	type NotesPageData struct {
		Message string
		Notes   []*NoteDTO
	}
	dtos := make([]*NoteDTO, len(notes))
	for i := range notes {
		dtos[i] = MapNote(notes[i])
	}
	message := p.ByName("message")
	data := NotesPageData{message, dtos}

	err = tmpl.ExecuteTemplate(rw, "main", data)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}
}

func (a App) ShowUpdateNotePage(rw http.ResponseWriter, r *http.Request, p httprouter.Params) {
	var noteID int64
	var err error
	idParam := strings.TrimPrefix(r.URL.Path, "/notes/")

	if idParam != "" {
		noteID, err = strconv.ParseInt(idParam, 10, 64)
		if err != nil {
			http.Error(rw, "параметр 'id' невалидный", http.StatusBadRequest)
			return
		}
	}

	note, _ := a.db.GetNoteById(a.ctx, noteID)

	tmpl := ParseTemplateFiles(rw, "updateNote.html")
	message := p.ByName("message")
	type UpdateNotePageData struct {
		Message string
		Note    *NoteUpdateDTO
	}
	data := UpdateNotePageData{message, MapNoteUpdate(note)}

	err = tmpl.ExecuteTemplate(rw, "updateNote", data)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}
}

func (a App) UpdateNote(rw http.ResponseWriter, r *http.Request, p httprouter.Params) {
	noteIDParam := strings.TrimSpace(r.FormValue("noteID"))
	noteName := strings.TrimSpace(r.FormValue("noteName"))
	noteDesc := strings.TrimSpace(r.FormValue("noteDesc"))
	hasDeadline := r.FormValue("deadlineDateCheckbox") == "on"
	deadline := strings.TrimSpace(r.FormValue("deadlineDatePicker"))
	isCompleted := r.FormValue("completedCheckbox") == "on"

	if noteName == "" || noteDesc == "" {
		p = append(p, httprouter.Param{Key: "message", Value: "Название и описание заметки не должны быть пустыми!"})
		r.URL.Path = "/notes/" + noteIDParam
		a.ShowUpdateNotePage(rw, r, p)
		return
	}

	if hasDeadline && deadline == "" {
		p = append(p, httprouter.Param{Key: "message", Value: "Укажите дату дедлайна!"})
		r.URL.Path = "/notes/" + noteIDParam
		a.ShowUpdateNotePage(rw, r, p)
		return
	}

	var noteID int64
	var err error
	noteID, err = strconv.ParseInt(noteIDParam, 10, 64)
	if err != nil {
		http.Error(rw, "параметр 'noteID' невалидный", http.StatusBadRequest)
		return
	}

	params := repository.UpdateNoteParams{
		Name:        noteName,
		Description: &noteDesc,
		IsCompleted: isCompleted,
		ID:          noteID,
	}

	if hasDeadline {
		parsedDeadline, _ := time.Parse(layoutISO, deadline)
		params.DeadlineAt = pgtype.Date{
			Time:             parsedDeadline,
			InfinityModifier: 0,
			Valid:            true,
		}
	}

	_, err = a.db.UpdateNote(a.ctx, params)
	if err != nil {
		p = append(p, httprouter.Param{Key: "message", Value: "Возникла ошибка при обновлении заметки!"})
		a.ShowCreateNotePage(rw, r, p)
		return
	}

	http.Redirect(rw, r, "/", http.StatusSeeOther)
}

func (a App) DeleteNote(rw http.ResponseWriter, r *http.Request, p httprouter.Params) {
	var noteID int64
	var err error
	idParam := strings.TrimPrefix(r.URL.Path, "/delete/")

	if idParam != "" {
		noteID, err = strconv.ParseInt(idParam, 10, 64)
		if err != nil {
			http.Error(rw, "параметр 'id' невалидный", http.StatusBadRequest)
			return
		}
	}

	_, err = a.db.DeleteNoteById(a.ctx, noteID)
	if err != nil {
		p = append(p, httprouter.Param{Key: "message", Value: "Возникла ошибка при удалении заметки!"})
		a.ShowMainPage(rw, r, p)
		return
	}

	http.Redirect(rw, r, "/", http.StatusSeeOther)
}

func (a App) ChangeStatusNote(rw http.ResponseWriter, r *http.Request, p httprouter.Params) {
	noteIDParam := strings.TrimSpace(r.FormValue("noteID"))
	noteChangeStatusToParam := strings.TrimSpace(r.FormValue("statusChangeTo"))

	var noteID int64
	var err error
	noteID, err = strconv.ParseInt(noteIDParam, 10, 64)
	if err != nil {
		http.Error(rw, "параметр 'noteID' невалидный", http.StatusBadRequest)
		return
	}

	isCompleted := false
	if noteChangeStatusToParam == "Завершить" {
		isCompleted = true
	}

	_, err = a.db.ChangeNoteStatus(a.ctx, repository.ChangeNoteStatusParams{
		IsCompleted: isCompleted,
		ID:          noteID,
	})
	if err != nil {
		p = append(p, httprouter.Param{Key: "message", Value: "Возникла ошибка при изменении статуса заметки!"})
		a.ShowMainPage(rw, r, p)
		return
	}

	http.Redirect(rw, r, "/", http.StatusSeeOther)
}

func (a App) ShowCreateNotePage(rw http.ResponseWriter, _ *http.Request, p httprouter.Params) {
	tmpl := ParseTemplateFiles(rw, "createNote.html")

	message := p.ByName("message")
	noteName := p.ByName("noteName")
	noteDesc := p.ByName("noteDesc")
	deadline := p.ByName("deadline")
	type CreateNotePageData struct {
		Message string
		Note    *NoteCreateDTO
	}
	data := CreateNotePageData{Message: message, Note: &NoteCreateDTO{noteName, noteDesc, deadline}}

	err := tmpl.ExecuteTemplate(rw, "createNote", data)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}
}

func (a App) AuthNeeded(next httprouter.Handle) httprouter.Handle {
	return func(rw http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		token, err := utils.ReadCookie(Token, r)
		if err != nil {
			http.Redirect(rw, r, "/login", http.StatusUnauthorized)
			return
		}

		user, ok := a.cache[token]
		if !ok {
			http.Redirect(rw, r, "/login", http.StatusUnauthorized)
			return
		}

		ps = append(ps, httprouter.Param{Key: "userID", Value: strconv.FormatInt(user.ID, 10)})

		next(rw, r, ps)
	}
}

func (a App) Register(rw http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	login := strings.TrimSpace(r.FormValue("login"))
	password := strings.TrimSpace(r.FormValue("password"))
	confirmPassword := strings.TrimSpace(r.FormValue("confirmPassword"))

	if login == "" || password == "" || confirmPassword == "" {
		a.ShowRegisterPage(rw, "Все поля должны быть заполнены!")
		return
	}

	if password != confirmPassword {
		a.ShowRegisterPage(rw, "Пароли не совпадают!")
		return
	}

	_, err := a.db.CreateUser(a.ctx, repository.CreateUserParams{
		Login:    login,
		Password: utils.GetHashedString(password),
	})
	if err != nil {
		a.ShowRegisterPage(rw, fmt.Sprintf("Ошибка создания пользователя: %v", err))
		return
	}

	a.ShowLoginPage(rw, "Регистрация успешна!")
}

func (a App) CreateNewNote(rw http.ResponseWriter, r *http.Request, p httprouter.Params) {
	noteName := strings.TrimSpace(r.FormValue("noteName"))
	noteDesc := strings.TrimSpace(r.FormValue("noteDesc"))
	hasDeadline := r.FormValue("deadlineDateCheckbox") == "on"
	deadline := strings.TrimSpace(r.FormValue("deadlineDatePicker"))

	if noteName == "" || noteDesc == "" {
		p = append(p, httprouter.Param{Key: "message", Value: "Название и описание заметки не должны быть пустыми!"})
		p = append(p, httprouter.Param{Key: "noteName", Value: noteName})
		p = append(p, httprouter.Param{Key: "noteDesc", Value: noteDesc})
		p = append(p, httprouter.Param{Key: "deadline", Value: deadline})
		a.ShowCreateNotePage(rw, r, p)
		return
	}

	if hasDeadline && deadline == "" {
		p = append(p, httprouter.Param{Key: "message", Value: "Укажите дату дедлайна!"})
		p = append(p, httprouter.Param{Key: "noteName", Value: noteName})
		p = append(p, httprouter.Param{Key: "noteDesc", Value: noteDesc})
		a.ShowCreateNotePage(rw, r, p)
		return
	}

	var userID int64
	var err error
	userIDParam := p.ByName("userID")
	if userIDParam == "" {
		http.Error(rw, "требуется параметр 'userID'", http.StatusBadRequest)
		return
	}
	userID, err = strconv.ParseInt(userIDParam, 10, 64)
	if err != nil {
		http.Error(rw, "параметр 'userID' невалидный", http.StatusBadRequest)
		return
	}

	params := repository.CreateNoteParams{
		UserID:      userID,
		Name:        noteName,
		Description: &noteDesc,
	}
	if hasDeadline {
		parsedDeadline, _ := time.Parse(layoutISO, deadline)
		params.DeadlineAt = pgtype.Date{
			Time:             parsedDeadline,
			InfinityModifier: 0,
			Valid:            true,
		}
	}

	_, err = a.db.CreateNote(a.ctx, params)
	if err != nil {
		p = append(p, httprouter.Param{Key: "message", Value: "Возникла ошибка при создании заметки!"})
		a.ShowCreateNotePage(rw, r, p)
		return
	}

	http.Redirect(rw, r, "/", http.StatusSeeOther)
}

func NewApp(ctx context.Context, db *repository.Queries) *App {
	return &App{ctx, db, make(map[string]*repository.User)}
}
