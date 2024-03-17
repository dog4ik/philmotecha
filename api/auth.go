package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/dog4ik/philmotecha/db"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

type UserPayload struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type jwtPayload struct {
	UserId int32 `json:"user_id"`
	Exp    int64 `json:"exp"`
	jwt.RegisteredClaims
}

type AuthResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func extractUser(r *http.Request, queries *db.Queries) (db.Appuser, error) {
	user_id, err := GetAuthenticatedUser(r)
	if err != nil {
		return db.Appuser{}, fmt.Errorf("unauthorized")
	}

	user, err := queries.GetUserById(context.Background(), user_id)

	if err != nil {
		if err == pgx.ErrNoRows {
			return db.Appuser{}, fmt.Errorf("notfound")
		}
		return db.Appuser{}, fmt.Errorf("internalerror")
	}
	return user, nil
}

func GetAuthenticatedUser(r *http.Request) (int32, error) {
	auth := r.Header.Get("Authorization")

	if auth == "" {
		return 0, fmt.Errorf("Auth header is absent")
	}

	bearer := "Bearer "
	auth = auth[len(bearer):]
	user_id, err := VerifyAccessToken(auth)
	if err != nil {
		return 0, err
	}
	return user_id, nil
}

type EnsureAdminAuth struct {
	handler http.Handler
	queries *db.Queries
}

func (ea *EnsureAdminAuth) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	user, err := extractUser(r, ea.queries)
	if err != nil {
		if err.Error() == "notfound" {
			error_response(w, "User not found", http.StatusNotFound)
			return
		}
		if err.Error() == "unauthorized" {
			error_response(w, "User is not authorized", http.StatusUnauthorized)
			return
		}
		error_response(w, "Unknown server error", http.StatusInternalServerError)
		return
	}
	if user.Role != db.UserRoleAdmin {
		error_response(w, fmt.Sprintf("%s rights are required, got %s", db.UserRoleAdmin, user.Role), http.StatusForbidden)
		return
	}

	ea.handler.ServeHTTP(w, r)
}

type EnsureAnyAuth struct {
	handler http.HandlerFunc
	queries *db.Queries
}

func (ea *EnsureAnyAuth) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	_, err := extractUser(r, ea.queries)
	if err != nil {
		if err.Error() == "notfound" {
			error_response(w, "User not found", http.StatusNotFound)
			return
		}
		if err.Error() == "unauthorized" {
			error_response(w, "User is not authorized", http.StatusUnauthorized)
			return
		}
		error_response(w, "Unknown server error", http.StatusInternalServerError)
		return
	}

	ea.handler.ServeHTTP(w, r)
}

func NewEnsureAnyAuth(queries *db.Queries) func(http.HandlerFunc) *EnsureAnyAuth {
	return func(handlerToWrap http.HandlerFunc) *EnsureAnyAuth {
		return &EnsureAnyAuth{handlerToWrap, queries}
	}
}

func NewEnsureAdminAuth(queries *db.Queries) func(http.HandlerFunc) *EnsureAdminAuth {
	return func(handlerToWrap http.HandlerFunc) *EnsureAdminAuth {
		return &EnsureAdminAuth{handlerToWrap, queries}
	}
}

func GenerateAccessToken(id int32) (string, error) {
	key := []byte(os.Getenv("ACCESS_TOKEN"))
	claims := jwtPayload{
		UserId: id,
		Exp:    time.Now().Add(30 * 24 * time.Hour).UnixMilli(),
	}
	if len(key) == 0 {
		return "", fmt.Errorf("access token env not set")
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, err := t.SignedString(key)
	if err != nil {
		return "", err
	}
	return s, nil
}

func GenerateRefreshToken(id int32) (string, error) {
	key := []byte(os.Getenv("REFRESH_TOKEN"))
	if len(key) == 0 {
		return "", fmt.Errorf("refresh token env not set")
	}
	claims := jwtPayload{
		UserId: id,
		Exp:    time.Now().Add(365 * 24 * time.Hour).UnixMilli(),
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, err := t.SignedString(key)
	if err != nil {
		return "", err
	}
	return s, nil
}

func VerifyAccessToken(token string) (int32, error) {
	res, err := jwt.ParseWithClaims(token, &jwtPayload{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(os.Getenv("ACCESS_TOKEN")), nil

	})
	if err != nil {
		return 0, fmt.Errorf("failed to parse token: %v", err)
	}
	if claims, ok := res.Claims.(*jwtPayload); ok && res.Valid {
		if claims.Exp-time.Now().UnixMilli() < 0 {
			return 0, fmt.Errorf("token expired")
		}
		return claims.UserId, nil
	}
	return 0, fmt.Errorf("error getting claims")
}

// AddUser
//
//	@Summary		Add an user
//	@Description	add by json user
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Param			user	body		api.UserPayload	true	"Add user"
//	@Success		201		{object}	db.Appuser
//	@Failure		400		{object}	api.ServerError
//	@Failure		500		{object}	api.ServerError
//	@Router			/add_user [post]
func (self *Database) InsertUser(w http.ResponseWriter, r *http.Request) {
	var payload UserPayload
	decode_err := json.NewDecoder(r.Body).Decode(&payload)
	if decode_err != nil {
		error_response(w, "Failed to decode request body", http.StatusBadRequest)
		return
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(payload.Password), 14)

	new_user, err := self.Queries.CreateUser(
		context.Background(),
		db.CreateUserParams{
			Username: payload.Username,
			Password: string(passwordHash),
			Role:     db.UserRoleUser,
		})

	if err != nil {
		log.Printf("ERROR: Failed to create user: %s", err)
		error_response(w, "Failed to create user", http.StatusInternalServerError)
		return
	}
	json_response(w, new_user, http.StatusCreated)
}

// Login
//
//	@Summary		Login an user
//	@Description	login user using password and username
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Param			user	body		api.UserPayload	true	"Login user"
//	@Success		200		{object}	api.AuthResponse
//	@Failure		400		{object}	api.ServerError
//	@Failure		404		{object}	api.ServerError
//	@Failure		500		{object}	api.ServerError
//	@Router			/login [post]
func (self *Database) LoginUser(w http.ResponseWriter, r *http.Request) {
	var payload UserPayload
	decode_err := json.NewDecoder(r.Body).Decode(&payload)
	if decode_err != nil {
		error_response(w, decode_err.Error(), http.StatusBadRequest)
		return
	}

	user, err := self.Queries.GetUserByUsername(context.Background(), payload.Username)

	if err != nil {
		if err == pgx.ErrNoRows {
			error_response(w, "User not found", http.StatusNotFound)
			return
		}

		log.Printf("ERROR: Database error: {%s}", err)
		error_response(w, "Internal database error", http.StatusInternalServerError)
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(payload.Password))
	if err != nil {
		error_response(w, "Passwords dont match", http.StatusInternalServerError)
		return
	}

	access_token, err := GenerateAccessToken(user.ID)
	if err != nil {
		log.Printf("ERROR: Failed to generate access token: {%s}", err)
		error_response(w, "Failed to generate access token", http.StatusInternalServerError)
		return
	}

	refresh_token, err := GenerateRefreshToken(user.ID)
	if err != nil {
		log.Printf("ERROR: Failed to generate refresh token: {%s}", err)
		error_response(w, "Failed to generate refresh token", http.StatusInternalServerError)
		return
	}

	response := AuthResponse{
		AccessToken:  access_token,
		RefreshToken: refresh_token,
	}

	json_response(w, response, http.StatusOK)
}
