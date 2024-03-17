-- name: GetActor :one
SELECT * FROM Actor
WHERE id = $1 LIMIT 1;

-- name: ListActors :many
SELECT 
    Actor.*,
    ARRAY_AGG(Movie.id) FILTER (WHERE Movie.id IS NOT NULL)::int[] AS movies
FROM 
    Actor
LEFT JOIN 
    MovieActor ON Actor.id = MovieActor.actor_id
LEFT JOIN 
    Movie ON MovieActor.movie_id = Movie.id
GROUP BY 
    Actor.id;

-- name: CreateActor :one
INSERT INTO Actor (
  name, birth, gender
) VALUES (
  $1, $2, $3
)
RETURNING *;

-- name: UpdateActor :exec
UPDATE Actor
  SET name = COALESCE(sqlc.narg('name'), name),
  birth = COALESCE(sqlc.narg('birth'), birth),
  gender = COALESCE(sqlc.narg('gender'), gender)
WHERE id = $1;

-- name: DeleteActor :one
DELETE FROM Actor
WHERE id = $1
RETURNING id;

-- name: CreateMovie :one
INSERT INTO Movie (
  title, description, release_date, rating
) VALUES (
  $1 ,$2, $3, $4 
)
RETURNING *;

-- name: UpdateMovie :exec
UPDATE Movie
  SET title = COALESCE(sqlc.narg('title'), title),
  description = COALESCE(sqlc.narg('description'), description),
  release_date = COALESCE(sqlc.narg('release_date'), release_date),
  rating = COALESCE(sqlc.narg('rating'), rating)
WHERE id = $1;

-- name: DeleteMovie :one
DELETE FROM Movie
WHERE id = $1
RETURNING id;

-- name: CreateUser :one
INSERT INTO AppUser (
  username, password, role
) VALUES (
  $1 ,$2, $3
)
RETURNING *;

-- name: GetUserByUsername :one
SELECT * FROM AppUser 
WHERE username = $1;

-- name: GetUserById :one
SELECT * FROM AppUser 
WHERE id = $1;

-- name: ListMoviesAsc :many
SELECT * FROM Movie
ORDER BY @property::text ASC;

-- name: ListMoviesDesc :many
SELECT * FROM Movie
ORDER BY @property::text DESC;

-- name: SearchMovie :many
SELECT Movie.*
FROM Movie 
LEFT JOIN 
    MovieActor ON Movie.id = MovieActor.movie_id
LEFT JOIN 
    Actor ON MovieActor.actor_id = Actor.id
WHERE 
  title @@ to_tsquery($1)
OR
  LOWER(actor.name) LIKE LOWER($2);;

-- name: ActorMovies :many
SELECT m.* 
FROM Movie m
LEFT JOIN MovieActor ma ON m.movie_id = ma.movie_id
WHERE ma.actor_id = 1;

-- name: CreateMovieActor :exec
INSERT INTO MovieActor (
  movie_id, actor_id
) VALUES (
  $1, $2
);


-- name: ClearDatabase :exec
DROP TABLE AppUser;
DROP TABLE Actor;
DROP TABLE Movie;
