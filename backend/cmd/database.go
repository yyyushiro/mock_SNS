package main

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Post struct {
	PostId    uuid.UUID
	UserId    uuid.UUID
	Body      string
	CreatedAt time.Time
	LikedByMe bool
	LikeCount int
}

type Like struct {
	LikerId   uuid.UUID `json:"liker_id"`
	PostId    uuid.UUID `json:"post_id"`
	CreatedAt time.Time `json:"created_at"`
}

func AddPost(sub uuid.UUID, body string, pool *pgxpool.Pool, ctx context.Context) (*Post, error) {
	var post Post
	err := pool.QueryRow(ctx,
		`INSERT INTO posts (user_id, body) VALUES ($1, $2)
     RETURNING id, user_id, body, created_at`,
		sub, body,
	).Scan(&post.PostId, &post.UserId, &post.Body, &post.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("Inserting post: %w", err)
	}
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
		SELECT id, body, created_at,
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
		if err := rows.Scan(&p.PostId, &p.Body, &p.CreatedAt, &p.LikedByMe); err != nil {
			return nil, fmt.Errorf("Scanning posts: %w", err)
		}

		p.LikeCount, err = getLikeCount(p.PostId, pool, ctx)
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
	rows, err := pool.Query(ctx, `SELECT id, body, created_at,
	EXISTS (
		SELECT 1 FROM likes
		WHERE likes.user_id = $1 AND likes.post_id = posts.id
	) FROM posts WHERE posts.user_id <> $1`, userId)
	if err != nil {
		return nil, fmt.Errorf("Getting post rows: %w", err)
	}
	defer rows.Close()
	var posts []Post
	for rows.Next() {
		var p Post
		if err := rows.Scan(&p.PostId, &p.Body, &p.CreatedAt, &p.LikedByMe); err != nil {
			return nil, fmt.Errorf("Scanning posts: %w", err)
		}

		p.LikeCount, err = getLikeCount(p.PostId, pool, ctx)
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
func LikePost(likerId, postId uuid.UUID, pool *pgxpool.Pool, ctx context.Context) (*Like, error) {
	// I'm not sure if I have to validate likerid and postid. I will do them later if needed.
	var l Like
	err := pool.QueryRow(ctx, `SELECT user_id, post_id, created_at FROM likes WHERE user_id = $1 AND post_id = $2`, likerId, postId).Scan(&l.LikerId, &l.PostId, &l.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			err := pool.QueryRow(ctx, `INSERT INTO likes (user_id, post_id) VALUES ($1, $2) RETURNING user_id, post_id, created_at`, likerId, postId).Scan(&l.LikerId, &l.PostId, &l.CreatedAt)
			if err != nil {
				return nil, fmt.Errorf("Scanned like: %w", err)
			}
			return &l, nil
		}
		return nil, fmt.Errorf("Scanned like: %w", err)
	}
	return &l, fmt.Errorf("User %d already liked post %d", l.LikerId, l.PostId)
}

// UndoLikePost undos the likes.
func UndoLikePost(likerId, postId uuid.UUID, pool *pgxpool.Pool, ctx context.Context) (*Like, error) {
	var l Like
	err := pool.QueryRow(ctx, `SELECT user_id, post_id, created_at FROM likes WHERE user_id = $1 AND post_id = $2`, likerId, postId).Scan(&l.LikerId, &l.PostId, &l.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("Scanned like: %w", err)
	}

	err = pool.QueryRow(ctx, `DELETE FROM likes WHERE user_id = $1 AND post_id = $2 RETURNING user_id, post_id, created_at`, likerId, postId).Scan(&l.LikerId, &l.PostId, &l.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("Deleted a like: %w", err)
	}

	return &l, nil
}

func getLikeCount(postId uuid.UUID, pool *pgxpool.Pool, ctx context.Context) (int, error) {
	var count int
	// While this query is related to multiple rows, since it eventually returns a single value, user QueryRow().
	err := pool.QueryRow(ctx, `SELECT COUNT(post_id) FROM likes WHERE post_id = $1`, postId).Scan(&count)
	if err != nil {
		return -1, fmt.Errorf("Counted likes: %w", err)
	}
	return count, nil
}
