package middleware

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

type contextKey string

const userIDContextKey contextKey = "userID"

// UserIDContextKey экспортируется для тестов (установка userID в контекст запроса)
var UserIDContextKey interface{} = userIDContextKey
const cookieName = "user_id"

// UserCookie middleware выдаёт симметрично подписанную cookie с идентификатором пользователя.
// Если cookie отсутствует — выдаётся новая cookie с новым user ID.
// Если cookie присутствует и подпись верна — user ID кладётся в контекст.
// Если cookie присутствует, но не содержит валидный user ID (подпись неверна) — новая cookie не выдаётся, в контекст кладётся пустая строка; хендлер GET /api/user/urls должен вернуть 401.
func UserCookie(secret string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(cookieName)
			var userID string
			cookiePresent := err == nil && cookie != nil && cookie.Value != ""
			if cookiePresent {
				userID, _ = verifySignedCookie(cookie.Value, secret)
			}
			if !cookiePresent {
				// Cookie отсутствует — выдаём новую
				userID = uuid.New().String()
				value := signCookie(userID, secret)
				http.SetCookie(w, &http.Cookie{
					Name:     cookieName,
					Value:    value,
					Path:     "/",
					HttpOnly: true,
				})
			}
			// Если cookie была, но невалидна — userID остаётся пустым, не выдаём новую cookie
			ctx := context.WithValue(r.Context(), userIDContextKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetUserID возвращает user ID из контекста. Пустая строка — пользователь не аутентифицирован (cookie была, но невалидна).
func GetUserID(ctx context.Context) string {
	v, _ := ctx.Value(userIDContextKey).(string)
	return v
}

func signCookie(userID string, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(userID))
	sig := hex.EncodeToString(mac.Sum(nil))
	return base64.URLEncoding.EncodeToString([]byte(userID)) + "." + sig
}

func verifySignedCookie(value string, secret string) (userID string, ok bool) {
	parts := strings.SplitN(value, ".", 2)
	if len(parts) != 2 {
		return "", false
	}
	decoded, err := base64.URLEncoding.DecodeString(parts[0])
	if err != nil {
		return "", false
	}
	userID = string(decoded)
	if userID == "" {
		return "", false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(userID))
	expected := hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expected), []byte(parts[1])) {
		return "", false
	}
	return userID, true
}
