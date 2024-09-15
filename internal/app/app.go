package app

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgtype"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"time"

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
	Completed          = "Завершено"
	Expired            = "Просрочено"
)

type NoteTypeClass string

const (
	Default NoteTypeClass = "text-white bg-primary"
	Success               = "text-white bg-success"
	Danger                = "text-white bg-danger"
)

type NoteDTO struct {
	ID          int64         `json:"id"`
	UserID      int64         `json:"user_id"`
	Name        string        `json:"name"`
	Description *string       `json:"description"`
	CreatedAt   string        `json:"created_at"`
	Type        NoteType      `json:"type"`
	TypeClass   NoteTypeClass `json:"type_class"`
}

type NoteUpdateDTO struct {
	ID          int64   `json:"id"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
	HasDeadline bool    `json:"has_deadline"`
	Deadline    string  `json:"deadline"`
	IsCompleted bool    `json:"is_completed"`
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
	layoutUS  = "January 2, 2006"
)

func MapNote(note *repository.Note) *NoteDTO {
	var noteType NoteType
	var noteTypeClass NoteTypeClass
	if note.IsCompleted {
		noteType = Completed
		noteTypeClass = Success
	} else if note.DeadlineAt.Valid && time.Now().After(note.DeadlineAt.Time) {
		noteType = Expired
		noteTypeClass = Danger
	} else {
		noteType = Active
		noteTypeClass = Default
	}
	return &NoteDTO{
		ID:          note.ID,
		UserID:      note.UserID,
		Name:        note.Name,
		Description: note.Description,
		CreatedAt:   note.CreatedAt.Time.Format(layoutISO),
		Type:        noteType,
		TypeClass:   noteTypeClass,
	}
}

func (a App) Routes(r *httprouter.Router) {
	r.ServeFiles("/public/*filepath", http.Dir("public"))
	r.GET("/", a.AuthNeeded(a.ShowMainPage))
	r.GET("/login", func(rw http.ResponseWriter, r *http.Request, p httprouter.Params) {
		a.ShowLoginPage(rw, "")
	})
	r.POST("/login", a.Login)
	r.GET("/logout", a.Logout)
	r.GET("/register", func(rw http.ResponseWriter, r *http.Request, p httprouter.Params) {
		a.ShowRegisterPage(rw, "")
	})
	r.POST("/register", a.Register)
	r.GET("/notes", a.AuthNeeded(a.ShowCreateNotePage))
	r.POST("/notes", a.AuthNeeded(a.CreateNewNote))
	r.GET("/notes/:page", a.AuthNeeded(a.ShowUpdateNotePage))
	r.POST("/search", a.AuthNeeded(a.PaginationMainPage))
	r.POST("/update", a.AuthNeeded(a.UpdateNote))
}

func (a App) ShowLoginPage(rw http.ResponseWriter, message string) {
	filePath := filepath.Join("public", "html", "login.html")

	tmpl, err := template.ParseFiles(filePath)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}
	data := PageData{message}

	err = tmpl.ExecuteTemplate(rw, "login", data)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}
}

func (a App) Login(rw http.ResponseWriter, r *http.Request, p httprouter.Params) {
	login := r.FormValue("login")
	password := r.FormValue("password")

	if login == "" || password == "" {
		a.ShowLoginPage(rw, "Необходимо указать логин и пароль!")
		return
	}
	user, err := a.db.GetUserByLoginAndPassword(a.ctx, repository.GetUserByLoginAndPasswordParams{Login: login,
		Password: utils.GetHashedString(password)})
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
	filePath := filepath.Join("public", "html", "register.html")

	tmpl, err := template.ParseFiles(filePath)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}

	data := PageData{message}

	err = tmpl.ExecuteTemplate(rw, "register", data)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}
}

func (a App) PaginationMainPage(rw http.ResponseWriter, r *http.Request, p httprouter.Params) {
	x := r.RequestURI
	log.Println(x)
}

func (a App) FilterMainPage(rw http.ResponseWriter, _ *http.Request, p httprouter.Params) {

}

func (a App) ShowMainPage(rw http.ResponseWriter, _ *http.Request, p httprouter.Params) {
	limit := int64(6)
	offset := int64(0)
	var userID int64
	var err error

	userIDParam := p.ByName("userID")
	if userIDParam == "" {
		http.Error(rw, "требуется параметр 'userID'", http.StatusBadRequest)
		return
	} else {
		userID, err = strconv.ParseInt(userIDParam, 10, 64)
		if err != nil {
			http.Error(rw, "параметр 'userID' невалидный", http.StatusBadRequest)
			return
		}
	}

	limitParam := p.ByName("limit")
	offsetParam := p.ByName("offset")
	if limitParam != "" {
		limit, err = strconv.ParseInt(limitParam, 10, 64)
		if err != nil {
			http.Error(rw, "параметр 'limit' невалидный", http.StatusBadRequest)
			return
		}
	}
	if offsetParam != "" {
		offset, err = strconv.ParseInt(offsetParam, 10, 64)
		if err != nil {
			http.Error(rw, "параметр 'offset' невалидный", http.StatusBadRequest)
			return
		}
	}

	notes, err := a.db.GetPageableNotesByUserId(a.ctx, repository.GetPageableNotesByUserIdParams{
		UserID: userID,
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}

	filePath := filepath.Join("public", "html", "main.html")

	tmpl, err := template.ParseFiles(filePath)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}

	type NotesPageData struct {
		Notes []*NoteDTO
	}
	dtos := make([]*NoteDTO, len(notes))
	for i, _ := range notes {
		dtos[i] = MapNote(notes[i])
	}
	data := NotesPageData{dtos}

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

	note, err := a.db.GetNoteById(a.ctx, noteID)

	message := p.ByName("message")
	type UpdateNotePageData struct {
		Message string
		Note    *NoteUpdateDTO
	}

	data := UpdateNotePageData{message, MapNoteUpdate(note)}

	filePath := filepath.Join("public", "html", "updateNote.html")

	tmpl, err := template.ParseFiles(filePath)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}

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

func (a App) ShowCreateNotePage(rw http.ResponseWriter, _ *http.Request, p httprouter.Params) {
	filePath := filepath.Join("public", "html", "createNote.html")

	tmpl, err := template.ParseFiles(filePath)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}

	message := p.ByName("message")
	data := PageData{message}

	err = tmpl.ExecuteTemplate(rw, "createNote", data)
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

func (a App) Register(rw http.ResponseWriter, r *http.Request, p httprouter.Params) {
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
		a.ShowCreateNotePage(rw, r, p)
		return
	}

	if hasDeadline && deadline == "" {
		p = append(p, httprouter.Param{Key: "message", Value: "Укажите дату дедлайна!"})
		a.ShowCreateNotePage(rw, r, p)
		return
	}

	var userID int64
	var err error
	userIDParam := p.ByName("userID")
	if userIDParam == "" {
		http.Error(rw, "требуется параметр 'userID'", http.StatusBadRequest)
		return
	} else {
		userID, err = strconv.ParseInt(userIDParam, 10, 64)
		if err != nil {
			http.Error(rw, "параметр 'userID' невалидный", http.StatusBadRequest)
			return
		}
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
