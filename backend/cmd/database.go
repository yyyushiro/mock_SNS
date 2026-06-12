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

func DeletePost(userId, postId uuid.UUID, pool *pgxpool.Pool, ctx context.Context) (err error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("starting transaction: %w", err)
	}
	defer func() {
		if err == nil {
			err = tx.Commit(ctx)
		}
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	if _, err = tx.Exec(ctx, "DELETE FROM likes WHERE post_id = $1", postId); err != nil {
		return fmt.Errorf("Deleting rows in likes: %w", err)
	}

	if _, err = tx.Exec(ctx, "DELETE FROM posts WHERE id = $1 AND user_id = $2", postId, userId); err != nil {
		return fmt.Errorf("Deleting a row in posts: %w", err)
	}
	return nil
}

func GetMyPosts(userId uuid.UUID, pool *pgxpool.Pool, ctx context.Context) ([]Post, error) {
	rows, err := pool.Query(ctx, `
	SELECT id, user_id, COALESCE(username, ''), body, created_at,
	EXISTS (
		SELECT 1 FROM likes
		WHERE likes.user_id = $1 AND likes.post_id = posts.id
	),
	(SELECT COUNT(*) FROM likes WHERE likes.post_id = posts.id)
	FROM posts
	WHERE user_id = $1`, userId)
	if err != nil {
		return nil, fmt.Errorf("Getting post rows: %w", err)
	}
	defer rows.Close()
	var posts []Post
	for rows.Next() {
		var p Post
		if err := rows.Scan(&p.Id, &p.UserId, &p.Username, &p.Body, &p.CreatedAt, &p.LikedByMe, &p.LikeCount); err != nil {
			return nil, fmt.Errorf("Scanning posts: %w", err)
		}
		posts = append(posts, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("Scanned posts: %w", err)
	}
	return posts, nil
}

func GetPublicPosts(userId uuid.UUID, pool *pgxpool.Pool, ctx context.Context) ([]Post, error) {
	rows, err := pool.Query(ctx, `
	SELECT id, user_id, COALESCE(username, ''), body, created_at,
	EXISTS (
		SELECT 1 FROM likes
		WHERE likes.user_id = $1 AND likes.post_id = posts.id
	),
	(SELECT COUNT(*) FROM likes WHERE likes.post_id = posts.id)
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
		if err := rows.Scan(&p.Id, &p.UserId, &p.Username, &p.Body, &p.CreatedAt, &p.LikedByMe, &p.LikeCount); err != nil {
			return nil, fmt.Errorf("Scanning posts: %w", err)
		}
		posts = append(posts, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("Scanned posts: %w", err)
	}
	return posts, nil
}

func GetFollowingPosts(userId uuid.UUID, pool *pgxpool.Pool, ctx context.Context) ([]Post, error) {
	rows, err := pool.Query(ctx, `
	SELECT id, user_id, COALESCE(username, ''), body, created_at,
	EXISTS (
		SELECT 1 FROM likes
		WHERE likes.user_id = $1 AND likes.post_id = posts.id
	),
	(SELECT COUNT(*) FROM likes WHERE likes.post_id = posts.id)
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
		if err := rows.Scan(&p.Id, &p.UserId, &p.Username, &p.Body, &p.CreatedAt, &p.LikedByMe, &p.LikeCount); err != nil {
			return nil, fmt.Errorf("Scanning posts: %w", err)
		}
		posts = append(posts, p)
	}
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

func UpdateUsername(userID uuid.UUID, username string, pool *pgxpool.Pool, ctx context.Context) (err error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if err == nil {
			err = tx.Commit(ctx)
		}
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	tag, err := tx.Exec(ctx, `UPDATE users SET username = $1 WHERE id = $2`, username, userID)
	if err != nil {
		return fmt.Errorf("updating username in users table: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("updating username in users table: %w", pgx.ErrNoRows)
	}

	_, err = tx.Exec(ctx, `UPDATE posts SET username = $1 WHERE user_id = $2`, username, userID)
	if err != nil {
		return fmt.Errorf("updating username in posts table: %w", err)
	}
	return nil
}

func DeleteUser(userID uuid.UUID, pool *pgxpool.Pool, ctx context.Context) (err error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("starting transaction: %w", err)
	}
	defer func() {
		if err == nil {
			err = tx.Commit(ctx)
		}
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	if _, err = tx.Exec(ctx, `DELETE FROM likes WHERE user_id = $1 OR post_id IN (SELECT id FROM posts WHERE user_id = $1)`, userID); err != nil {
		return fmt.Errorf("deleting likes: %w", err)
	}
	if _, err = tx.Exec(ctx, `DELETE FROM follows WHERE follower_id = $1 OR followee_id = $1`, userID); err != nil {
		return fmt.Errorf("deleting follows: %w", err)
	}
	if _, err = tx.Exec(ctx, `DELETE FROM posts WHERE user_id = $1`, userID); err != nil {
		return fmt.Errorf("deleting posts: %w", err)
	}
	if _, err = tx.Exec(ctx, `DELETE FROM users WHERE id = $1`, userID); err != nil {
		return fmt.Errorf("deleting user: %w", err)
	}
	return nil
}

// MarkEmailVerified flips email_verified to true for the given user.
// 0 rows affected means the userId from Redis has no matching Postgres row —
// a data-integrity breach that should never happen in normal operation.
func MarkEmailVerified(ctx context.Context, pool *pgxpool.Pool, userId uuid.UUID) error {
	tag, err := pool.Exec(ctx, `UPDATE users SET email_verified = true WHERE id = $1`, userId)
	if err != nil {
		return fmt.Errorf("marking email verified: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("marking email verified: no user row for id %s", userId)
	}
	return nil
}

// GetPasswordUserByEmail looks up a password-based account by its normalized email.
// Returns the user's id, bcrypt-hashed password, and email_verified flag.
// pgx.ErrNoRows is returned (wrapped) when no matching row exists.
func GetPasswordUserByEmail(ctx context.Context, pool *pgxpool.Pool, normalizedEmail string) (uuid.UUID, string, bool, error) {
	var id uuid.UUID
	var hashedPassword string
	var emailVerified bool
	err := pool.QueryRow(ctx,
		`SELECT id, hashed_password, email_verified FROM users WHERE email = $1`,
		normalizedEmail,
	).Scan(&id, &hashedPassword, &emailVerified)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.UUID{}, "", false, fmt.Errorf("get password user by email: %w", pgx.ErrNoRows)
		}
		return uuid.UUID{}, "", false, fmt.Errorf("get password user by email: %w", err)
	}
	return id, hashedPassword, emailVerified, nil
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
