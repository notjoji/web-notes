package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/notjoji/web-notes/internal/repository"
	"github.com/pkg/errors"
)

func ReadCookie(name string, r *http.Request) (value string, err error) {
	if name == "" {
		return value, errors.New("reading empty cookie")
	}
	cookie, err := r.Cookie(name)
	if err != nil {
		return value, err
	}
	str := cookie.Value
	return url.QueryUnescape(str)
}

func GetAuthToken(user *repository.User) string {
	time64 := time.Now().Unix()
	timeInt := strconv.FormatInt(time64, 10)
	token := user.Login + user.Password + timeInt
	return GetHashedString(token)
}

func GetHashedString(str string) string {
	h := sha256.New()
	h.Write([]byte(str))
	return hex.EncodeToString(h.Sum(nil))
}
