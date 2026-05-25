package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

func sqlNullStringToString(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}

type Post struct {
	Id        uuid.UUID
	UserId    uuid.UUID
	Username  string
	Body      string
	CreatedAt time.Time
	LikedByMe bool
	LikeCount int
}

type User struct {
	Id        uuid.UUID
	Username  sql.NullString
	CreatedAt time.Time
}

func AddPost(sub uuid.UUID, body string, pool *pgxpool.Pool, ctx context.Context) (*Post, error) {
	var post Post
	var username sql.NullString
	err := pool.QueryRow(ctx,
		`INSERT INTO posts (user_id, body, username)
		 SELECT $1, $2, u.username
		 FROM users u
		 WHERE u.id = $1
		 RETURNING id, user_id, username, body, created_at`,
		sub, body,
	).Scan(&post.Id, &post.UserId, &username, &post.Body, &post.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("Inserting post: %w", err)
	}
	post.Username = sqlNullStringToString(username)
	return &post, nil
}

func DeletePost(userId, postId uuid.UUID, pool *pgxpool.Pool, ctx context.Context) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("starting transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, "DELETE FROM likes WHERE post_id = $1", postId); err != nil {
		return fmt.Errorf("Deleting rows in likes: %w", err)
	}

	if _, err := tx.Exec(ctx, "DELETE FROM posts WHERE id = $1 AND user_id = $2", postId, userId); err != nil {
		return fmt.Errorf("Deleting a row in posts: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("Committing transaction: %w", err)
	}
	return nil
}

func GetMyPosts(userId uuid.UUID, pool *pgxpool.Pool, ctx context.Context) ([]Post, error) {
	rows, err := pool.Query(ctx, `
		SELECT id, user_id, username, body, created_at,
			EXISTS (
				SELECT 1 FROM likes
				WHERE likes.user_id = $1 AND likes.post_id = posts.id
			)
		FROM posts WHERE user_id = $1`, userId)
	if err != nil {
		return nil, fmt.Errorf("Getting post rows: %w", err)
	}
	defer rows.Close()
	var posts []Post
	for rows.Next() {
		var p Post
		var username sql.NullString
		if err := rows.Scan(&p.Id, &p.UserId, &username, &p.Body, &p.CreatedAt, &p.LikedByMe); err != nil {
			return nil, fmt.Errorf("Scanning posts: %w", err)
		}
		p.Username = sqlNullStringToString(username)

		p.LikeCount, err = aggregateLikeCount(p.Id, pool, ctx)
		if err != nil {
			return nil, err
		}

		posts = append(posts, p)
	}
	// This error block catches when the iteration finishes abnormally.
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("Scanned posts: %w", err)
	}
	return posts, nil
}

func GetPublicPosts(userId uuid.UUID, pool *pgxpool.Pool, ctx context.Context) ([]Post, error) {
	rows, err := pool.Query(ctx, `SELECT id, user_id, username, body, created_at,
	EXISTS (
		SELECT 1 FROM likes
		WHERE likes.user_id = $1 AND likes.post_id = posts.id
	) 
	FROM posts 
	WHERE posts.user_id <> $1
	AND NOT EXISTS (
	  SELECT 1 FROM follows f
	  WHERE f.follower_id = $1 AND f.followee_id = posts.user_id)`, userId)
	if err != nil {
		return nil, fmt.Errorf("Getting post rows: %w", err)
	}
	defer rows.Close()
	var posts []Post
	for rows.Next() {
		var p Post
		var username sql.NullString
		if err := rows.Scan(&p.Id, &p.UserId, &username, &p.Body, &p.CreatedAt, &p.LikedByMe); err != nil {
			return nil, fmt.Errorf("Scanning posts: %w", err)
		}
		p.Username = sqlNullStringToString(username)

		p.LikeCount, err = aggregateLikeCount(p.Id, pool, ctx)
		if err != nil {
			return nil, err
		}

		posts = append(posts, p)
	}
	// This error block catches when the iteration finishes abnormally.
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("Scanned posts: %w", err)
	}
	return posts, nil
}

func GetFollowingPosts(userId uuid.UUID, pool *pgxpool.Pool, ctx context.Context) ([]Post, error) {
	rows, err := pool.Query(ctx, `SELECT id, user_id, username, body, created_at,
	EXISTS (
		SELECT 1 FROM likes
		WHERE likes.user_id = $1 AND likes.post_id = posts.id
	) 
	FROM posts 
	WHERE posts.user_id <> $1
	AND EXISTS (
	  SELECT 1 FROM follows f
	  WHERE f.follower_id = $1 AND f.followee_id = posts.user_id)`, userId)
	if err != nil {
		return nil, fmt.Errorf("Getting post rows: %w", err)
	}
	defer rows.Close()
	var posts []Post
	for rows.Next() {
		var p Post
		var username sql.NullString
		if err := rows.Scan(&p.Id, &p.UserId, &username, &p.Body, &p.CreatedAt, &p.LikedByMe); err != nil {
			return nil, fmt.Errorf("Scanning posts: %w", err)
		}
		p.Username = sqlNullStringToString(username)

		p.LikeCount, err = aggregateLikeCount(p.Id, pool, ctx)
		if err != nil {
			return nil, err
		}

		posts = append(posts, p)
	}
	// This error block catches when the iteration finishes abnormally.
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("Scanned posts: %w", err)
	}
	return posts, nil
}

// LikePost increments likes with the check of no duplicates.
func LikePost(likerId, postId uuid.UUID, pool *pgxpool.Pool, ctx context.Context) error {
	// I'm not sure if I have to validate likerid and postid. I will do them later if needed.
	tag, err := pool.Exec(ctx, `SELECT 1 FROM likes WHERE user_id = $1 AND post_id = $2`, likerId, postId)
	if err != nil {
		return fmt.Errorf("getting a like: %w", err)
	}
	if tag.RowsAffected() != 0 {
		return fmt.Errorf("user %v already liked post %v", likerId, postId)
	}

	_, err = pool.Exec(ctx, `INSERT INTO likes (user_id, post_id) VALUES ($1, $2)`, likerId, postId)
	if err != nil {
		return fmt.Errorf("inserting a like: %w", err)
	}
	return nil
}

// UndoLikePost undos the likes.
func UndoLikePost(likerId, postId uuid.UUID, pool *pgxpool.Pool, ctx context.Context) error {
	tag, err := pool.Exec(ctx, `DELETE FROM likes WHERE user_id = $1 AND post_id = $2`, likerId, postId)
	if err != nil {
		return fmt.Errorf("deleting a like: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("user %v has not liked post %v", likerId, postId)
	}
	return nil
}

func aggregateLikeCount(postId uuid.UUID, pool *pgxpool.Pool, ctx context.Context) (int, error) {
	var count int
	// While this query is related to multiple rows, since it eventually returns a single value, user QueryRow().
	err := pool.QueryRow(ctx, `SELECT COUNT(post_id) FROM likes WHERE post_id = $1`, postId).Scan(&count)
	if err != nil {
		return -1, fmt.Errorf("Counted likes: %w", err)
	}
	return count, nil
}

func AddFollow(followerId, followeeId uuid.UUID, pool *pgxpool.Pool, ctx context.Context) error {
	_, err := pool.Exec(ctx, "INSERT INTO follows (follower_id, followee_id) VALUES ($1, $2)", followerId, followeeId)
	if err != nil {
		return fmt.Errorf("Inserting into follows: %w", err)
	}

	return nil
}

func DeleteFollow(followerId, followeeId uuid.UUID, pool *pgxpool.Pool, ctx context.Context) error {
	_, err := pool.Exec(ctx, "DELETE FROM follows WHERE follower_id = $1 AND followee_id = $2", followerId, followeeId)
	if err != nil {
		return fmt.Errorf("Deleting from follows: %w", err)
	}

	return nil
}

func GetUserInfo(userId uuid.UUID, pool *pgxpool.Pool, ctx context.Context) (*User, error) {
	var user User
	err := pool.QueryRow(ctx, "SELECT id, username, created_at FROM users WHERE id = $1", userId).Scan(&user.Id, &user.Username, &user.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("Getting userinfo: %w", err)
	}

	return &user, nil
}

func UpdateMyUsername(userID uuid.UUID, username string, pool *pgxpool.Pool, ctx context.Context) error {
	tag, err := pool.Exec(ctx, `UPDATE users SET username = $1 WHERE id = $2`, username, userID)
	if err != nil {
		return fmt.Errorf("updating username in users table: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("updating username in users table: %w", pgx.ErrNoRows)
	}

	_, err = pool.Exec(ctx, `UPDATE posts SET username = $1 WHERE user_id = $2`, username, userID)
	if err != nil {
		return fmt.Errorf("updating username in posts table: %w", err)
	}
	return nil
}

// InsertPasswordUser creates a new user row with the given normalized email and
// bcrypt-hashed password. It returns the generated UUID on success.
//
// The caller is responsible for normalizing the email before passing it here;
// the unique index on LOWER(email) enforces uniqueness at the DB level.
// A PostgreSQL unique-violation (23505) is surfaced as a sentinel error so the
// HTTP layer can return 409 without logging a full stack trace.
func InsertPasswordUser(ctx context.Context, pool *pgxpool.Pool, email, hashedPassword string) (uuid.UUID, error) {
	var id uuid.UUID
	err := pool.QueryRow(ctx,
		`INSERT INTO users (email, hashed_password) VALUES ($1, $2) RETURNING id`,
		email, hashedPassword,
	).Scan(&id)
	if err != nil {
		var pgErr *pgconn.PgError
		// 23505 = unique_violation; the LOWER(email) index was hit.
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return uuid.UUID{}, fmt.Errorf("email already registered")
		}
		return uuid.UUID{}, fmt.Errorf("inserting password user: %w", err)
	}
	return id, nil
}
