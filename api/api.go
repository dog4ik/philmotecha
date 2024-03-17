package api

import (
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"strconv"

	"github.com/dog4ik/philmotecha/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type Database struct {
	Queries db.Queries
}

func json_response(w http.ResponseWriter, body any, status int) {
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(status)
	err := json.NewEncoder(w).Encode(body)
	log.Printf("TRACE: Success response with status: %d", status)
	if err != nil {
		log.Printf("RESPONSE: Failed to serialize response body: {%s}", err)
	}
}

type ServerError struct {
	Message string `json:"message"`
}

func error_response(w http.ResponseWriter, message string, status int) {
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(status)
	server_error := ServerError{Message: message}
	err := json.NewEncoder(w).Encode(server_error)
	log.Printf("TRACE: Error response(%d): %s", status, message)
	if err != nil {
		log.Fatalf("Failed to serialize response body: %s\n", err)
	}
}

// ListActors lists all existing actors
//
//	@Summary		List actors
//	@Description	get actors
//	@Tags			actors
//	@Produce		json
//	@Success		200	{array}		db.ListActorsRow
//	@Failure		401	{object}	api.ServerError
//	@Failure		500	{object}	api.ServerError
//	@Router			/list_actors [get]
//	@Security		JwtAuth
func (self *Database) ListActors(w http.ResponseWriter, r *http.Request) {
	actors, err := self.Queries.ListActors(r.Context())
	if err != nil {
		log.Printf("ERROR: Failed to list all actors {%s}", err)
		error_response(w, "failed to list all actors", http.StatusInternalServerError)
		return
	}
	if actors == nil {
		json_response(w, []string{}, http.StatusOK)
		return
	}
	json_response(w, actors, http.StatusOK)
}

// List movies lists all existing movies
//
//	@Summary		List movies
//	@Description	get movies
//	@Tags			movies
//	@Produce		json
//	@Param			sort_type	query		string	false	"Sort direction"	Enums(desc, asc)			default(desc)
//	@Param			sort_by		query		string	false	"Sort property"		Enums(rating, title, date)	default(rating)
//	@Success		200			{array}		db.Actor
//	@Failure		401			{object}	api.ServerError
//	@Failure		500			{object}	api.ServerError
//	@Router			/list_movies [get]
//	@Security		JwtAuth
func (self *Database) ListMovies(w http.ResponseWriter, r *http.Request) {
	sort_by := r.URL.Query().Get("sort_by")
	sort_type := r.URL.Query().Get("sort_type")
	switch sort_by {
	case "rating", "":
		sort_by = "rating"
	case "title":
		sort_by = "title"
	case "date":
		sort_by = "release_date"
	default:
		error_response(w, fmt.Sprintf("parameter %s is not recognized", sort_by), http.StatusBadRequest)
		return
	}

	switch sort_type {
	case "", "desc":
		sort_type = "ASC"
	case "asc":
		sort_type = "ASC"
	default:
		error_response(w, fmt.Sprintf("parameter %s is not recognized", sort_by), http.StatusBadRequest)
		return
	}

	var err error
	var movies []db.Movie
	if sort_type == "DESC" {
		movies, err = self.Queries.ListMoviesDesc(r.Context(), sort_by)
	} else if sort_type == "ASC" {
		movies, err = self.Queries.ListMoviesAsc(r.Context(), sort_by)
	}

	if err != nil {
		log.Printf("ERROR: Failed to list all actors {%s}", err)
		error_response(w, "Server database error", http.StatusInternalServerError)
		return
	}

	if movies == nil {
		json_response(w, []string{}, http.StatusOK)
		return
	}
	json_response(w, movies, http.StatusOK)
}

// AddActor
//
//	@Summary		Add an actor
//	@Description	add by json actor
//	@Tags			actors
//	@Accept			json
//	@Produce		json
//	@Param			actor	body		db.CreateActorParams	true	"Add actor"
//	@Success		201		{object}	db.Actor
//	@Failure		400		{object}	api.ServerError
//	@Failure		401		{object}	api.ServerError
//	@Failure		403		{object}	api.ServerError
//	@Failure		500		{object}	api.ServerError
//	@Router			/add_actor [post]
//	@Security		JwtAuth
func (self *Database) InsertActor(w http.ResponseWriter, r *http.Request) {
	var payload db.CreateActorParams
	decode_err := json.NewDecoder(r.Body).Decode(&payload)
	if decode_err != nil {
		error_response(w, decode_err.Error(), http.StatusBadRequest)
		return
	}

	new_actor, err := self.Queries.CreateActor(r.Context(), payload)
	if err != nil {
		log.Printf("ERROR: Failed to create Actor: {%s}", err)
		error_response(w, "Failed to create actor", http.StatusInternalServerError)
		return
	}
	json_response(w, new_actor, http.StatusCreated)
}

type NewMovieParams struct {
	Title       string         `json:"title"`
	Description pgtype.Text    `json:"description"`
	ReleaseDate pgtype.Date    `json:"release_date"`
	Rating      pgtype.Numeric `json:"rating"`
	Actors      []int32        `json:"actors"`
}

// AddMovie
//
//	@Summary		Add an movie
//	@Description	add by json movie
//	@Tags			movies
//	@Accept			json
//	@Produce		json
//	@Param			movie	body		api.NewMovieParams	true	"Add movie"
//	@Success		201		{object}	db.Movie
//	@Failure		400		{object}	api.ServerError
//	@Failure		401		{object}	api.ServerError
//	@Failure		403		{object}	api.ServerError
//	@Failure		500		{object}	api.ServerError
//	@Router			/add_movie [post]
//	@Security		JwtAuth
func (self *Database) InsertMovie(w http.ResponseWriter, r *http.Request) {
	var payload NewMovieParams
	decode_err := json.NewDecoder(r.Body).Decode(&payload)
	if decode_err != nil {
		error_response(w, decode_err.Error(), http.StatusBadRequest)
		return
	}

	if len(payload.Description.String) > 1000 {
		error_response(w, "Description length must be less then 1000 characters", http.StatusBadRequest)
		return
	}
	if len(payload.Title) < 1 || len(payload.Title) > 150 {
		error_response(w, "Movie title length must be between 1 and 150 characters", http.StatusBadRequest)
		return
	}

	new_movie, err := self.Queries.CreateMovie(r.Context(), db.CreateMovieParams{
		Title:       payload.Title,
		Description: payload.Description,
		ReleaseDate: payload.ReleaseDate,
		Rating:      payload.Rating,
	})
	if err != nil {
		log.Printf("ERROR: Failed to create Movie: {%s}", err)
		error_response(w, "failed to insert Movie", http.StatusInternalServerError)
		return
	}
	for _, actor_id := range payload.Actors {
		err = self.Queries.CreateMovieActor(
			r.Context(),
			db.CreateMovieActorParams{
				MovieID: new_movie.ID,
				ActorID: actor_id,
			})
		if err != nil {
			log.Printf("ERROR: Failed to create Movie: {%s}", err)
			error_response(w, "failed to connect actor with movie", http.StatusInternalServerError)
			return
		}
	}
	json_response(w, new_movie, http.StatusCreated)
}

type ActorPayload struct {
	Name   pgtype.Text    `json:"name" example:"Daryl Dixon"`
	Birth  pgtype.Date    `json:"birth"`
	Gender *db.GenderType `json:"gender"`
}

// UpdateActor
//
//	@Summary		Update an actor
//	@Description	Update actor by json payload
//	@Tags			actors
//	@Accept			json
//	@Produce		json
//	@Param			actor	body	api.ActorPayload	true	"Update actor"
//	@Param			id	path	int	true	"Actor ID"
//	@Success		200
//	@Failure		400	{object}	api.ServerError
//	@Failure		401	{object}	api.ServerError
//	@Failure		403	{object}	api.ServerError
//	@Failure		404	{object}	api.ServerError
//	@Failure		500	{object}	api.ServerError
//	@Router			/update_actor/{id} [patch]
//	@Security		JwtAuth
func (self *Database) UpdateActor(w http.ResponseWriter, r *http.Request) {
	path_id := r.PathValue("id")
	id, err := strconv.Atoi(path_id)
	if err != nil || len(path_id) == 0 || id < 0 {
		error_response(w, "Could not parse id param", http.StatusBadRequest)
		return
	}

	var payload ActorPayload
	decode_err := json.NewDecoder(r.Body).Decode(&payload)
	if decode_err != nil {
		error_response(w, decode_err.Error(), http.StatusBadRequest)
		return
	}
	var gender db.GenderType
	if payload.Gender != nil {
		gender = *payload.Gender
	}
	err = self.Queries.UpdateActor(r.Context(),
		db.UpdateActorParams{
			ID:     int32(id),
			Name:   payload.Name,
			Birth:  payload.Birth,
			Gender: db.NullGenderType{GenderType: gender, Valid: payload.Gender != nil},
		})

	if err != nil {
		if err == pgx.ErrNoRows {
			error_response(w, err.Error(), http.StatusNotFound)
			return
		}
		log.Printf("ERROR: Failed to update actor: {%s}", err)
		error_response(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

type MoviePayload struct {
	Description  pgtype.Text    `json:"description" maxLength:"1000" example:"Boring movie about planents"`
	Rating       pgtype.Numeric `json:"rating" minimum:"0" maximum:"10"`
	Release_date pgtype.Date    `json:"release_date"`
	Title        pgtype.Text    `json:"title" minLength:"1" maxLength:"150" example:"Inception"`
}

func (c MoviePayload) Validate() error {
	if c.Description.Valid && len(c.Description.String) > 1000 {
		return fmt.Errorf("Description length must be below 1000 characters")
	}
	if c.Title.Valid && (len(c.Title.String) > 150 || len(c.Title.String) < 1) {
		return fmt.Errorf("Title length must be between 1 and 10")
	}
	if c.Rating.Valid && (c.Rating.Int.Cmp(big.NewInt(10)) < 1 || c.Rating.Int.Cmp(big.NewInt(0)) > -1) {
		return fmt.Errorf("Rating must be a number between 0 and 10")
	}
	return nil
}

// UpdateMovie
//
//	@Summary		Update a movie
//	@Description	Update by json movie
//	@Tags			movies
//	@Accept			json
//	@Produce		json
//	@Param			movie	body	api.MoviePayload	true	"Update movie"
//	@Param			id	path	int	true	"Movie ID"
//	@Success		200
//	@Failure		400	{object}	api.ServerError
//	@Failure		401	{object}	api.ServerError
//	@Failure		403	{object}	api.ServerError
//	@Failure		404	{object}	api.ServerError
//	@Failure		500	{object}	api.ServerError
//	@Router			/update_movie/{id} [patch]
//	@Security		JwtAuth
func (self *Database) UpdateMovie(w http.ResponseWriter, r *http.Request) {
	var payload MoviePayload
	path_id := r.PathValue("id")
	id, err := strconv.Atoi(path_id)
	if err != nil || len(path_id) == 0 || id < 0 {
		error_response(w, "Could not parse id param", http.StatusBadRequest)
		return
	}
	decode_err := json.NewDecoder(r.Body).Decode(&payload)
	if decode_err != nil {
		error_response(w, "Could not decode payload", http.StatusBadRequest)
		return
	}
	err = payload.Validate()
	if err != nil {
		error_response(w, fmt.Sprintf("Could not validate payload: %s", err), http.StatusBadRequest)
		return
	}
	err = self.Queries.UpdateMovie(r.Context(), db.UpdateMovieParams{
		ID:          int32(id),
		Title:       payload.Title,
		Description: payload.Description,
	})

	if err != nil {
		if err == pgx.ErrNoRows {
			error_response(w, err.Error(), http.StatusNotFound)
			return
		}
		log.Printf("ERROR: Failed to update movie: {%s}", err)
		error_response(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// DeleteActor
//
//	@Summary		Delete an actor
//	@Description	Delete by json actor
//	@Tags			actors
//	@Accept			json
//	@Produce		json
//	@Param			id	path	int	true	"Actor ID"
//	@Success		200
//	@Failure		400	{object}	api.ServerError
//	@Failure		401	{object}	api.ServerError
//	@Failure		403	{object}	api.ServerError
//	@Failure		404	{object}	api.ServerError
//	@Failure		500	{object}	api.ServerError
//	@Router			/delete_actor/{id} [delete]
//	@Security		JwtAuth
func (self *Database) DeleteActor(w http.ResponseWriter, r *http.Request) {
	path_id := r.PathValue("id")
	id, parse_err := strconv.Atoi(path_id)
	if parse_err != nil {
		error_response(w, parse_err.Error(), http.StatusBadRequest)
		return
	}
	_, err := self.Queries.DeleteActor(r.Context(), int32(id))
	if err != nil {
		if err == pgx.ErrNoRows {
			error_response(w, err.Error(), http.StatusNotFound)
			return
		}
		log.Printf("ERROR: Failed to remove actor: {%s}", err)
		error_response(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// DeleteMovie
//
//	@Summary		Delete an movie
//	@Description	Delete by json movie
//	@Tags			movies
//	@Accept			json
//	@Produce		json
//	@Param			id	path	int	true	"Movie ID"
//	@Success		200
//	@Failure		400	{object}	api.ServerError
//	@Failure		401	{object}	api.ServerError
//	@Failure		403	{object}	api.ServerError
//	@Failure		404	{object}	api.ServerError
//	@Failure		500	{object}	api.ServerError
//	@Router			/delete_movie/{id} [delete]
//	@Security		JwtAuth
func (self *Database) DeleteMovie(w http.ResponseWriter, r *http.Request) {
	path_id := r.PathValue("id")
	id, parse_err := strconv.Atoi(path_id)
	if parse_err != nil {
		error_response(w, parse_err.Error(), http.StatusBadRequest)
		return
	}
	_, err := self.Queries.DeleteMovie(r.Context(), int32(id))

	if err != nil {
		if err == pgx.ErrNoRows {
			error_response(w, err.Error(), http.StatusNotFound)
			return
		}
		log.Printf("ERROR: Failed to remove movie: {%s}", err)
		error_response(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// SearchMovie
//
//	@Summary		Search lookup by fragment of movie's name or fragment of actor's name
//	@Description	Search by query
//	@Tags			movies
//	@Accept			json
//	@Produce		json
//	@Param			q query	string	true	"search by query"
//	@Success		200	{array}		db.Movie
//	@Failure		400	{object}	api.ServerError
//	@Failure		401	{object}	api.ServerError
//	@Failure		500	{object}	api.ServerError
//	@Router			/search [get]
//	@Security		JwtAuth
func (self *Database) SearchMovie(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	movies, err := self.Queries.SearchMovie(r.Context(), db.SearchMovieParams{
		ToTsquery: fmt.Sprintf("%s:*", query),
		Lower:     fmt.Sprintf("%%%s%%", query),
	})

	if err != nil {
		if err == pgx.ErrNoRows {
			emptySlice := []int{}
			json_response(w, emptySlice, http.StatusOK)
			return
		}
		log.Printf("ERROR: Failed to search movie: %s", err)
		error_response(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if movies == nil {
		json_response(w, []string{}, http.StatusOK)
		return
	}
	json_response(w, movies, http.StatusOK)
}

func (self *Database) ClearDb(w http.ResponseWriter, r *http.Request) {
	err := self.Queries.ClearDatabase(r.Context())
	if err != nil {
		log.Fatalf("Failed to clear db %s", err)
	}
	log.Printf("Cleared database\n")
}
