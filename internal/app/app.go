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

func (a App) Routes(r *httprouter.Router) {
	r.ServeFiles("/public/*filepath", http.Dir("public"))
	r.GET("/", a.AuthNeeded(a.ShowNotesPage))
	r.GET("/login", func(rw http.ResponseWriter, r *http.Request, p httprouter.Params) {
		a.ShowLoginPage(rw, "")
	})
	r.POST("/login", a.Login)
	r.GET("/logout", a.Logout)
	r.GET("/register", func(rw http.ResponseWriter, r *http.Request, p httprouter.Params) {
		a.ShowRegisterPage(rw, "")
	})
	r.POST("/register", a.Register)
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

func (a App) ShowNotesPage(rw http.ResponseWriter, _ *http.Request, p httprouter.Params) {
	limit := int64(5)
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

	filePath := filepath.Join("public", "html", "notes.html")

	tmpl, err := template.ParseFiles(filePath)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}

	type NotesPageData struct {
		Notes []*repository.Note
	}
	data := NotesPageData{notes}

	err = tmpl.ExecuteTemplate(rw, "notes", data)
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

func NewApp(ctx context.Context, db *repository.Queries) *App {
	return &App{ctx, db, make(map[string]*repository.User)}
}
