import { useEffect, useState } from "react"
import { Link, useNavigate, useSearchParams } from "react-router-dom"
import {
    getMyPosts,
    getPublicPosts,
    likePost,
    logout,
    unlikePost,
    type Post,
} from "../apis/API.ts"

type FeedView = "mine" | "public"

export default function TimeLinePage() {
    const navigate = useNavigate()
    const [searchParams, setSearchParams] = useSearchParams()
    const feedView: FeedView =
        searchParams.get("view") === "feeds" ? "public" : "mine"

    function setFeedView(next: FeedView) {
        if (next === "public") {
            setSearchParams({ view: "feeds" })
        } else {
            setSearchParams({})
        }
    }

    const [posts, setPosts] = useState<Post[]>([])
    const [loading, setLoading] = useState(true)
    const [error, setError] = useState<string | null>(null)
    const [pendingLikePostId, setPendingLikePostId] = useState<string | null>(
        null,
    )
    const [likeToggleError, setLikeToggleError] = useState<string | null>(null)
    const [loggingOut, setLoggingOut] = useState(false)
    const [logoutError, setLogoutError] = useState<string | null>(null)

    useEffect(() => {
        let cancelled = false
        setLoading(true)
        setError(null)
        const fetchPosts =
            feedView === "mine" ? getMyPosts() : getPublicPosts()
        fetchPosts
            .then((data) => {
                if (!cancelled) setPosts(data)
            })
            .catch((err: unknown) => {
                if (cancelled) return
                const message =
                    err instanceof Error ? err.message : "Failed to load posts"
                const isUnauthorized =
                    err instanceof Error &&
                    err.message.includes("Response status: 401")
                setError(
                    isUnauthorized ? "Not signed in." : message,
                )
            })
            .finally(() => {
                if (!cancelled) setLoading(false)
            })
        return () => {
            cancelled = true
        }
    }, [feedView])

    async function toggleLike(post: Post) {
        if (pendingLikePostId !== null) return
        setLikeToggleError(null)
        setPendingLikePostId(post.id)
        try {
            if (post.liked_by_me) {
                await unlikePost(post.id)
            } else {
                await likePost(post.id)
            }
            setPosts((prev) =>
                prev.map((p) => {
                    if (p.id !== post.id) return p
                    const nextLiked = !p.liked_by_me
                    const delta = nextLiked ? 1 : -1
                    return {
                        ...p,
                        liked_by_me: nextLiked,
                        like_count: Math.max(0, p.like_count + delta),
                    }
                }),
            )
        } catch (err: unknown) {
            const message =
                err instanceof Error ? err.message : "Could not update like"
            setLikeToggleError(message)
        } finally {
            setPendingLikePostId(null)
        }
    }

    async function handleLogout() {
        setLogoutError(null)
        setLoggingOut(true)
        try {
            await logout()
            navigate("/", { replace: true })
        } catch (err: unknown) {
            const message =
                err instanceof Error ? err.message : "Could not log out"
            setLogoutError(message)
        } finally {
            setLoggingOut(false)
        }
    }

    const centerTitle = feedView === "mine" ? "My Posts" : "Feeds"
    const switchLabel = feedView === "mine" ? "Feeds" : "My Posts"
    const switchTo: FeedView = feedView === "mine" ? "public" : "mine"

    return (
        <div className="timeline-page">
            <header className="timeline-header">
                <div className="timeline-header__start">
                    <button
                        type="button"
                        className="btn btn--primary"
                        onClick={() => navigate("/post")}
                    >
                        Add Posts
                    </button>
                </div>
                <div className="timeline-header__center">
                    <h1 className="timeline-title">{centerTitle}</h1>
                    {!loading && !error && (
                        <button
                            type="button"
                            className="btn btn--primary"
                            onClick={() => setFeedView(switchTo)}
                        >
                            {switchLabel}
                        </button>
                    )}
                </div>
                <div className="timeline-header__end">
                    {!loading && !error && (
                        <button
                            type="button"
                            className="btn btn--primary"
                            disabled={loggingOut}
                            onClick={() => {
                                void handleLogout()
                            }}
                        >
                            {loggingOut ? "Logging out…" : "Log out"}
                        </button>
                    )}
                </div>
            </header>

            <div className="timeline-main">
                {loading && <p>Loading…</p>}
                {error && (
                    <>
                        <p className="app-alert" role="alert">
                            {error}
                        </p>
                        {error === "Not signed in." && (
                            <p>
                                <Link to="/">Sign in</Link>. If login still fails,
                                set Google OAuth redirect and{" "}
                                <code>REDIRECT_URI</code> to{" "}
                                <code>
                                    http://localhost:5173/api/auth/callback/google
                                </code>{" "}
                                so cookies are stored for this dev server.
                            </p>
                        )}
                    </>
                )}
                {!loading &&
                    !error &&
                    (posts.length === 0 ? (
                        <>
                            {logoutError && (
                                <p className="app-alert" role="alert">
                                    {logoutError}
                                </p>
                            )}
                            <p className="timeline-empty">
                                {feedView === "mine"
                                    ? "No posts yet."
                                    : "No posts from others yet."}
                            </p>
                        </>
                    ) : (
                        <>
                            {logoutError && (
                                <p className="app-alert" role="alert">
                                    {logoutError}
                                </p>
                            )}
                            {likeToggleError && (
                                <p className="app-alert" role="alert">
                                    {likeToggleError}
                                </p>
                            )}
                            <ul className="post-list">
                                {posts.map((post) => {
                                    const busy = pendingLikePostId === post.id
                                    const liked = post.liked_by_me
                                    const formattedDate =
                                        post.created_at
                                            ? new Date(
                                                  post.created_at,
                                              ).toLocaleString(undefined, {
                                                  dateStyle: "medium",
                                                  timeStyle: "short",
                                              })
                                            : ""
                                    const bodyTrimmed = post.body.trim()
                                    return (
                                        <li key={post.id} className="post-card">
                                            {formattedDate ? (
                                                <time
                                                    className="post-card__date"
                                                    dateTime={post.created_at}
                                                >
                                                    {formattedDate}
                                                </time>
                                            ) : (
                                                <span className="post-card__date post-card__date--placeholder">
                                                    —
                                                </span>
                                            )}
                                            {bodyTrimmed ? (
                                                <div className="post-card-body">
                                                    {post.body}
                                                </div>
                                            ) : (
                                                <div className="post-card-body post-card-body--placeholder">
                                                    —
                                                </div>
                                            )}
                                            <div className="post-card__footer">
                                                <button
                                                    type="button"
                                                    disabled={busy}
                                                    aria-pressed={liked}
                                                    className={`btn-like${
                                                        liked
                                                            ? " btn-like--active"
                                                            : ""
                                                    }`}
                                                    aria-label={
                                                        liked
                                                            ? `Unlike, ${post.like_count} likes`
                                                            : `Like, ${post.like_count} likes`
                                                    }
                                                    onClick={() =>
                                                        toggleLike(post)
                                                    }
                                                >
                                                    ♥{" "}
                                                    <span aria-hidden="true">
                                                        {post.like_count}
                                                    </span>
                                                </button>
                                            </div>
                                        </li>
                                    )
                                })}
                            </ul>
                        </>
                    ))}
            </div>
        </div>
    )
}
