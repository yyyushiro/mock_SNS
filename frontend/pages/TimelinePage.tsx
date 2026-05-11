import { useEffect, useState } from "react"
import { Link, useNavigate, useSearchParams } from "react-router-dom"
import {
    deletePost,
    followUser,
    getFollowingPosts,
    getMyPosts,
    getPublicPosts,
    likePost,
    unfollowUser,
    unlikePost,
    type Post,
} from "../apis/API.ts"

type FeedView = "mine" | "public" | "following"

function postDisplayAuthor(post: Post): string {
    const name = post.username.trim()
    return name !== "" ? name : post.user_id
}

function feedViewFromSearch(view: string | null): FeedView {
    if (view === "feeds") return "public"
    if (view === "following") return "following"
    return "mine"
}

export default function TimeLinePage() {
    const navigate = useNavigate()
    const [searchParams, setSearchParams] = useSearchParams()
    const feedView: FeedView = feedViewFromSearch(searchParams.get("view"))

    function setFeedView(next: FeedView) {
        if (next === "public") {
            setSearchParams({ view: "feeds" })
        } else if (next === "following") {
            setSearchParams({ view: "following" })
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
    const [pendingDeletePostId, setPendingDeletePostId] = useState<
        string | null
    >(null)
    const [deletePostError, setDeletePostError] = useState<string | null>(
        null,
    )
    const [pendingFollowPostId, setPendingFollowPostId] = useState<
        string | null
    >(null)
    const [followActionError, setFollowActionError] = useState<string | null>(
        null,
    )

    useEffect(() => {
        let cancelled = false
        setLoading(true)
        setError(null)
        const fetchPosts =
            feedView === "mine"
                ? getMyPosts()
                : feedView === "public"
                  ? getPublicPosts()
                  : getFollowingPosts()
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

    useEffect(() => {
        setDeletePostError(null)
        setFollowActionError(null)
    }, [feedView])

    async function toggleLike(post: Post) {
        if (pendingLikePostId !== null || pendingFollowPostId !== null) return
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

    async function handleFollowPost(post: Post) {
        if (pendingFollowPostId !== null || pendingLikePostId !== null) return
        setFollowActionError(null)
        setPendingFollowPostId(post.id)
        try {
            await followUser(post.user_id)
            const data = await getPublicPosts()
            setPosts(data)
        } catch (err: unknown) {
            const message =
                err instanceof Error ? err.message : "Could not follow user"
            setFollowActionError(message)
        } finally {
            setPendingFollowPostId(null)
        }
    }

    async function handleUnfollowPost(post: Post) {
        if (pendingFollowPostId !== null || pendingLikePostId !== null) return
        setFollowActionError(null)
        setPendingFollowPostId(post.id)
        try {
                await unfollowUser(post.user_id)
            const data = await getFollowingPosts()
            setPosts(data)
        } catch (err: unknown) {
            const message =
                err instanceof Error ? err.message : "Could not unfollow user"
            setFollowActionError(message)
        } finally {
            setPendingFollowPostId(null)
        }
    }

    async function handleDeletePost(post: Post) {
        if (pendingDeletePostId !== null) return
        setDeletePostError(null)
        setPendingDeletePostId(post.id)
        try {
            await deletePost(post.id)
            setPosts((prev) => prev.filter((p) => p.id !== post.id))
        } catch (err: unknown) {
            const message =
                err instanceof Error ? err.message : "Could not delete post"
            setDeletePostError(message)
        } finally {
            setPendingDeletePostId(null)
        }
    }

    const centerTitle =
        feedView === "mine"
            ? "My Posts"
            : feedView === "public"
              ? "Feeds"
              : "Following"

    const feedTabs: { id: FeedView; label: string }[] = [
        { id: "mine", label: "My Posts" },
        { id: "public", label: "Feeds" },
        { id: "following", label: "Following" },
    ]

    const emptyMessage =
        feedView === "mine"
            ? "No posts yet."
            : feedView === "public"
              ? "No posts from others yet."
              : "No posts from people you follow yet."

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
                <div className="timeline-header__center timeline-header__center--stacked">
                    <h1 className="timeline-title">{centerTitle}</h1>
                    {!loading && !error && (
                        <nav
                            className="timeline-feed-nav"
                            aria-label="Choose feed"
                        >
                            {feedTabs.map((tab) => (
                                <button
                                    key={tab.id}
                                    type="button"
                                    className={
                                        feedView === tab.id
                                            ? "btn-feed-tab btn-feed-tab--active"
                                            : "btn-feed-tab"
                                    }
                                    aria-current={
                                        feedView === tab.id ? "page" : undefined
                                    }
                                    onClick={() => setFeedView(tab.id)}
                                >
                                    {tab.label}
                                </button>
                            ))}
                        </nav>
                    )}
                </div>
                <div className="timeline-header__end">
                    {!loading && !error && (
                        <Link
                            to="/mypage"
                            className="btn btn--primary"
                        >
                            My page
                        </Link>
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
                            <p className="timeline-empty">{emptyMessage}</p>
                        </>
                    ) : (
                        <>
                            {likeToggleError && (
                                <p className="app-alert" role="alert">
                                    {likeToggleError}
                                </p>
                            )}
                            {followActionError &&
                                (feedView === "public" ||
                                    feedView === "following") && (
                                    <p className="app-alert" role="alert">
                                        {followActionError}
                                    </p>
                                )}
                            {deletePostError && feedView === "mine" && (
                                <p className="app-alert" role="alert">
                                    {deletePostError}
                                </p>
                            )}
                            <ul className="post-list">
                                {posts.map((post) => {
                                    const busy = pendingLikePostId === post.id
                                    const deleting =
                                        pendingDeletePostId === post.id
                                    const followBusy =
                                        pendingFollowPostId === post.id
                                    const rowBusy =
                                        busy ||
                                        deleting ||
                                        followBusy ||
                                        pendingFollowPostId !== null ||
                                        pendingLikePostId !== null
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
                                    const footerExtra =
                                        feedView === "public" ||
                                        feedView === "following"
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
                                            {footerExtra && (
                                                <div className="post-card__author">
                                                    {postDisplayAuthor(post)}
                                                </div>
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
                                            <div
                                                className={
                                                    feedView === "mine"
                                                        ? "post-card__footer post-card__footer--mine"
                                                        : footerExtra
                                                          ? "post-card__footer post-card__footer--discover"
                                                          : "post-card__footer"
                                                }
                                            >
                                                <button
                                                    type="button"
                                                    disabled={rowBusy}
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
                                                {feedView === "public" && (
                                                    <button
                                                        type="button"
                                                        className="btn-follow"
                                                        disabled={rowBusy}
                                                        aria-label={`Follow author of post`}
                                                        onClick={() =>
                                                            void handleFollowPost(
                                                                post,
                                                            )
                                                        }
                                                    >
                                                        {followBusy
                                                            ? "Following…"
                                                            : "Follow"}
                                                    </button>
                                                )}
                                                {feedView === "following" && (
                                                    <button
                                                        type="button"
                                                        className="btn-unfollow"
                                                        disabled={rowBusy}
                                                        aria-label={`Unfollow author of post`}
                                                        onClick={() =>
                                                            void handleUnfollowPost(
                                                                post,
                                                            )
                                                        }
                                                    >
                                                        {followBusy
                                                            ? "Unfollowing…"
                                                            : "Unfollow"}
                                                    </button>
                                                )}
                                                {feedView === "mine" && (
                                                    <button
                                                        type="button"
                                                        className="btn-delete"
                                                        disabled={
                                                            deleting ||
                                                            busy ||
                                                            pendingDeletePostId !==
                                                                null
                                                        }
                                                        aria-label={`Delete post ${post.id}`}
                                                        onClick={() =>
                                                            void handleDeletePost(
                                                                post,
                                                            )
                                                        }
                                                    >
                                                        {deleting
                                                            ? "Deleting…"
                                                            : "Delete"}
                                                    </button>
                                                )}
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
