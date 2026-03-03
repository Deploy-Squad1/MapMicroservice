package middlewares

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const UserIDKey contextKey = "userID"
const RoleKey contextKey = "role"

func RequireRole(allowedRoles ...string) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("access_token")
			if err != nil {
				http.Error(w, "Unauthorized: missing token", http.StatusUnauthorized)
				return
			}

			token, err := jwt.Parse(cookie.Value, func(token *jwt.Token) (interface{}, error) {
				if token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
					return nil, fmt.Errorf("unexpected signing method")
				}

				secret := os.Getenv("JWT_SECRET")
				if secret == "" {
					log.Println("Auth Error: JWT_SECRET environment variable is not set")
					return nil, fmt.Errorf("secret not configured")
				}
				return []byte(secret), nil
			})

			if err != nil || !token.Valid {
				http.Error(w, "Unauthorized: invalid or expired token", http.StatusUnauthorized)
				return
			}

			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				http.Error(w, "Unauthorized: invalid claims format", http.StatusUnauthorized)
				return
			}

			userRole, ok := claims["role"].(string)
			if !ok {
				http.Error(w, "Forbidden: no role found in token", http.StatusForbidden)
				return
			}

			hasAccess := false
			if len(allowedRoles) == 0 {
				hasAccess = true
			} else {
				for _, allowed := range allowedRoles {
					if userRole == allowed {
						hasAccess = true
						break
					}
				}
			}

			if !hasAccess {
				http.Error(w, "Forbidden: you don't have the necessary role", http.StatusForbidden)
				return
			}

			rawID, ok := claims["user_id"].(string)
			if !ok {
				log.Println("Auth Error: user_id is missing or not a string in token")
				http.Error(w, "Unauthorized: invalid user_id format", http.StatusUnauthorized)
				return
			}

			userID, err := strconv.Atoi(rawID)
			if err != nil {
				log.Printf("Auth Error: failed to convert user_id to int: %v\n", err)
				http.Error(w, "Unauthorized: user_id is not a valid number", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), UserIDKey, userID)
			ctx = context.WithValue(ctx, RoleKey, userRole)

			next.ServeHTTP(w, r.WithContext(ctx))
		}
	}
}
