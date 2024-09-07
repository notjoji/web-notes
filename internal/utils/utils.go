package utils

import (
	"crypto/md5"
	"encoding/hex"
	"github.com/notjoji/web-notes/internal/repository"
	"github.com/pkg/errors"
	"net/http"
	"net/url"
	"strconv"
	"time"
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
	hash := md5.Sum([]byte(str))
	return hex.EncodeToString(hash[:])
}
