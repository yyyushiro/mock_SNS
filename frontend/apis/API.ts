export type Post = {
    id: string
    user_id: string
    username: string
    body: string
    created_at: string
    liked_by_me: boolean
    like_count: number
}

type PostWire = {
    id?: unknown
    user_id?: unknown
    username?: unknown
    body?: unknown
    created_at?: unknown
    liked_by_me?: unknown
    like_count?: unknown
}

function postRowFromJson(row: PostWire): Post {
    return {
        id: String(row.id ?? ""),
        user_id: String(row.user_id ?? ""),
        username: String(row.username ?? ""),
        body: String(row.body ?? ""),
        created_at: String(row.created_at ?? ""),
        liked_by_me: Boolean(row.liked_by_me),
        like_count: Number(row.like_count ?? 0),
    }
}

function postsFromJson(data: unknown): Post[] {
    if (!Array.isArray(data)) return []
    return data.map((item) => postRowFromJson(item as PostWire))
}

const REFRESH_URL = "/api/auth/refresh"

function requestUrl(input: RequestInfo | URL): string {
    if (typeof input === "string") return input
    if (input instanceof URL) return `${input.pathname}${input.search}`
    return input.url
}

function skipsRefreshRetry(url: string): boolean {
    return (
        url.includes("/api/auth/refresh") || url.includes("/api/auth/logout")
    )
}

function redirectToLogin(): void {
    window.location.href = "/"
}

let refreshInFlight: Promise<boolean> | null = null

async function refreshSession(): Promise<boolean> {
    if (refreshInFlight) return refreshInFlight
    refreshInFlight = (async () => {
        const res = await fetch(REFRESH_URL, {
            method: "POST",
            credentials: "include",
        })
        return res.ok
    })()
    try {
        return await refreshInFlight
    } finally {
        refreshInFlight = null
    }
}

async function apiFetch(
    input: RequestInfo | URL,
    init?: RequestInit,
): Promise<Response> {
    const mergedInit: RequestInit = { ...init, credentials: "include" }
    let response = await fetch(input, mergedInit)

    if (response.status !== 401 || skipsRefreshRetry(requestUrl(input))) {
        return response
    }

    const refreshed = await refreshSession()
    if (!refreshed) {
        redirectToLogin()
        throw new Error(`Response status: ${response.status}`)
    }

    response = await fetch(input, mergedInit)
    if (response.status === 401) {
        redirectToLogin()
        throw new Error(`Response status: ${response.status}`)
    }
    return response
}

export async function getMyPosts(): Promise<Post[]> {
    const response = await apiFetch("/api/user/me/posts", {
        method: "GET",
    })
    if (!response.ok) {
        throw new Error(`Response status: ${response.status}`)
    }
    const contentType = response.headers.get("content-type")
    if (!contentType || !contentType.includes("application/json")) {
        throw new TypeError("Oops, we haven't got JSON!")
    }
    const data: unknown = await response.json()
    return postsFromJson(data)
}

export async function getPublicPosts(): Promise<Post[]> {
    const response = await apiFetch("/api/user/me/posts/public", {
        method: "GET",
    })
    if (!response.ok) {
        throw new Error(`Response status: ${response.status}`)
    }
    const contentType = response.headers.get("content-type")
    if (!contentType || !contentType.includes("application/json")) {
        throw new TypeError("Oops, we haven't got JSON!")
    }
    const data: unknown = await response.json()
    return postsFromJson(data)
}

export async function getFollowingPosts(): Promise<Post[]> {
    const response = await apiFetch("/api/user/me/posts/following", {
        method: "GET",
    })
    if (!response.ok) {
        throw new Error(`Response status: ${response.status}`)
    }
    const contentType = response.headers.get("content-type")
    if (!contentType || !contentType.includes("application/json")) {
        throw new TypeError("Oops, we haven't got JSON!")
    }
    const data: unknown = await response.json()
    return postsFromJson(data)
}

export async function followUser(userId: string): Promise<void> {
    const response = await apiFetch(
        `/api/user/${encodeURIComponent(userId)}/follow`,
        {
            method: "POST",
        },
    )
    if (!response.ok) {
        throw new Error(`Response status: ${response.status}`)
    }
}

export async function unfollowUser(userId: string): Promise<void> {
    const response = await apiFetch(
        `/api/user/${encodeURIComponent(userId)}/follow`,
        {
            method: "DELETE",
        },
    )
    if (!response.ok) {
        throw new Error(`Response status: ${response.status}`)
    }
}

export async function makePost(body: string): Promise<Post> {
    const response = await apiFetch("/api/posts", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ body }),
    })
    if (!response.ok) {
        throw new Error(`Response status: ${response.status}`)
    }
    const contentType = response.headers.get("content-type")
    if (!contentType || !contentType.includes("application/json")) {
        throw new TypeError("Oops, we haven't got JSON!")
    }
    const raw: unknown = await response.json()
    return postRowFromJson(raw as PostWire)
}

export async function likePost(postId: string): Promise<void> {
    const response = await apiFetch(
        `/api/posts/${encodeURIComponent(postId)}/likes`,
        {
            method: "POST",
        },
    )
    if (!response.ok) {
        throw new Error(`Response status: ${response.status}`)
    }
}

export async function unlikePost(postId: string): Promise<void> {
    const response = await apiFetch(
        `/api/posts/${encodeURIComponent(postId)}/likes`,
        {
            method: "DELETE",
        },
    )
    if (!response.ok) {
        throw new Error(`Response status: ${response.status}`)
    }
}

export async function deletePost(postId: string): Promise<void> {
    const response = await apiFetch(
        `/api/posts/${encodeURIComponent(postId)}`,
        {
            method: "DELETE",
        },
    )
    if (!response.ok) {
        throw new Error(`Response status: ${response.status}`)
    }
}

export class RegisterError extends Error {
    readonly reason: "email_already_registered"
    constructor(reason: "email_already_registered") {
        super(reason)
        this.name = "RegisterError"
        this.reason = reason
    }
}

export async function register(email: string, password: string): Promise<void> {
    const response = await fetch("/api/auth/register", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ email, password }),
    })
    if (response.status === 409) throw new RegisterError("email_already_registered")
    if (!response.ok) throw new Error(`Response status: ${response.status}`)
}

export class LoginError extends Error {
    readonly reason: "invalid_credentials" | "email_not_verified"
    constructor(reason: "invalid_credentials" | "email_not_verified") {
        super(reason)
        this.name = "LoginError"
        this.reason = reason
    }
}

export async function login(email: string, password: string): Promise<void> {
    const response = await fetch("/api/auth/login", {
        method: "POST",
        credentials: "include",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ email, password }),
    })
    if (response.status === 401) throw new LoginError("invalid_credentials")
    if (response.status === 403) throw new LoginError("email_not_verified")
    if (!response.ok) throw new Error(`Response status: ${response.status}`)
}

export async function resendVerification(email: string): Promise<void> {
    await fetch("/api/auth/resend-verification", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ email }),
    })
}

export async function logout(): Promise<void> {
    const response = await fetch("/api/auth/logout", {
        method: "POST",
        credentials: "include",
    })
    if (!response.ok) {
        throw new Error(`Response status: ${response.status}`)
    }

    const refreshResponse = await fetch("/api/auth/refresh", {
        method: "DELETE",
        credentials: "include",
    })
    if (!refreshResponse.ok) {
        throw new Error(`Response status: ${refreshResponse.status}`)
    }
}

export type MyUserInfo = {
    id: string
    username: string
    created_at: string
}

type MyUserInfoWire = {
    id?: unknown
    username?: unknown
    created_at?: unknown
}

function myUserInfoFromJson(row: MyUserInfoWire): MyUserInfo {
    return {
        id: String(row.id ?? ""),
        username: String(row.username ?? ""),
        created_at: String(row.created_at ?? ""),
    }
}

export async function getMyInfo(): Promise<MyUserInfo> {
    const response = await apiFetch("/api/user/me", {
        method: "GET",
    })
    if (!response.ok) {
        throw new Error(`Response status: ${response.status}`)
    }
    const contentType = response.headers.get("content-type")
    if (!contentType || !contentType.includes("application/json")) {
        throw new TypeError("Oops, we haven't got JSON!")
    }
    const raw: unknown = await response.json()
    return myUserInfoFromJson(raw as MyUserInfoWire)
}

export async function updateMyUsername(username: string): Promise<MyUserInfo> {
    const response = await apiFetch("/api/user/me/name", {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ username }),
    })
    if (!response.ok) {
        throw new Error(`Response status: ${response.status}`)
    }
    const contentType = response.headers.get("content-type")
    if (!contentType || !contentType.includes("application/json")) {
        throw new TypeError("Oops, we haven't got JSON!")
    }
    const raw: unknown = await response.json()
    return myUserInfoFromJson(raw as MyUserInfoWire)
}
