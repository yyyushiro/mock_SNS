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

export async function getMyPosts(): Promise<Post[]> {
    const response = await fetch("/api/user/me/posts", {
        method: "GET",
        credentials: "include",
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
    const response = await fetch("/api/user/me/posts/public", {
        method: "GET",
        credentials: "include",
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
    const response = await fetch("/api/user/me/posts/following", {
        method: "GET",
        credentials: "include",
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
    const response = await fetch(
        `/api/user/${encodeURIComponent(userId)}/follow`,
        {
            method: "POST",
            credentials: "include",
        },
    )
    if (!response.ok) {
        throw new Error(`Response status: ${response.status}`)
    }
}

export async function unfollowUser(userId: string): Promise<void> {
    const response = await fetch(
        `/api/user/${encodeURIComponent(userId)}/follow`,
        {
            method: "DELETE",
            credentials: "include",
        },
    )
    if (!response.ok) {
        throw new Error(`Response status: ${response.status}`)
    }
}

export async function makePost(body: string): Promise<Post> {
    const response = await fetch("/api/posts", {
        method: "POST",
        credentials: "include",
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
    const response = await fetch(
        `/api/posts/${encodeURIComponent(postId)}/likes`,
        {
            method: "POST",
            credentials: "include",
        },
    )
    if (!response.ok) {
        throw new Error(`Response status: ${response.status}`)
    }
}

export async function unlikePost(postId: string): Promise<void> {
    const response = await fetch(
        `/api/posts/${encodeURIComponent(postId)}/likes`,
        {
            method: "DELETE",
            credentials: "include",
        },
    )
    if (!response.ok) {
        throw new Error(`Response status: ${response.status}`)
    }
}

export async function deletePost(postId: string): Promise<void> {
    const response = await fetch(
        `/api/posts/${encodeURIComponent(postId)}`,
        {
            method: "DELETE",
            credentials: "include",
        },
    )
    if (!response.ok) {
        throw new Error(`Response status: ${response.status}`)
    }
}

export async function logout(): Promise<void> {
    const response = await fetch("/api/auth/logout", {
        method: "POST",
        credentials: "include",
    })
    if (!response.ok) {
        throw new Error(`Response status: ${response.status}`)
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
    const response = await fetch("/api/user/me", {
        method: "GET",
        credentials: "include",
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
    const response = await fetch("/api/user/me", {
        method: "PATCH",
        credentials: "include",
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
