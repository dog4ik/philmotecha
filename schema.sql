CREATE TYPE gender_type AS ENUM ('male', 'female', 'arch');
CREATE TYPE user_role AS ENUM ('admin', 'user');

CREATE TABLE AppUser (
    id SERIAL PRIMARY KEY,
    username VARCHAR(100) NOT NULL UNIQUE,
    password VARCHAR(255) NOT NULL,
    role user_role NOT NULL
);

CREATE TABLE Actor (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    gender gender_type NOT NULL,
    birth DATE
);

CREATE TABLE Movie (
    id SERIAL PRIMARY KEY,
    title VARCHAR(150) NOT NULL,
    description TEXT,
    release_date DATE,
    rating NUMERIC(3,1) CHECK (rating >= 0 AND rating <= 10)
);

CREATE TABLE MovieActor (
    movie_id INT REFERENCES Movie(id) ON DELETE CASCADE,
    actor_id INT REFERENCES Actor(id) ON DELETE CASCADE,
    PRIMARY KEY (movie_id, actor_id)
);

CREATE INDEX title_idx
ON Movie
USING GIN ((to_tsvector('english',title)));

CREATE OR REPLACE FUNCTION reindex_movies()
RETURNS TRIGGER AS $$
BEGIN
    REINDEX CONCURRENTLY INDEX title_idx;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER movie_table_trigger
AFTER INSERT OR UPDATE OR DELETE
ON Movie
FOR EACH STATEMENT
EXECUTE FUNCTION reindex_movies();
