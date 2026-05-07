export type Post = {
    id: string
    body: string
    liked_by_me: boolean
    like_count: number
}

function postsFromJson(data: unknown): Post[] {
    if (!Array.isArray(data)) return []
    return data.map((item) => {
        const row = item as {
            id?: unknown
            body?: unknown
            liked_by_me?: unknown
            like_count?: unknown
        }
        return {
            id: String(row.id ?? ""),
            body: String(row.body ?? ""),
            liked_by_me: Boolean(row.liked_by_me),
            like_count: Number(row.like_count),
        }
    })
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
    const row = raw as {
        id?: unknown
        body?: unknown
        liked_by_me?: unknown
        like_count?: unknown
    }
    return {
        id: String(row.id ?? ""),
        body: String(row.body ?? ""),
        liked_by_me: Boolean(row.liked_by_me),
        like_count: Number(row.like_count ?? 0),
    }
}

function assertJsonResponse(response: Response): void {
    const contentType = response.headers.get("content-type")
    if (!contentType || !contentType.includes("application/json")) {
        throw new TypeError("Oops, we haven't got JSON!")
    }
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
    assertJsonResponse(response)
    await response.json()
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
    assertJsonResponse(response)
    await response.json()
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
