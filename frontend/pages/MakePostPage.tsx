import { useState, type FormEvent } from "react"
import { useNavigate } from "react-router-dom"
import { makePost } from "../apis/API.ts"

export default function MakePostPage() {
    const [text, setText] = useState("")
    const [loading, setLoading] = useState(false)
    const [error, setError] = useState<string | null>(null)
    const navigate = useNavigate()

    async function handleSubmit(e: FormEvent) {
        e.preventDefault()
        const body = text.trim()
        if (body === "") return
        setLoading(true)
        setError(null)
        try {
            await makePost(body)
            navigate("/timeline")
        } catch (err: unknown) {
            const message =
                err instanceof Error ? err.message : "Failed to create post"
            const isUnauthorized =
                err instanceof Error &&
                err.message.includes("Response status: 401")
            setError(isUnauthorized ? "Not signed in." : message)
        } finally {
            setLoading(false)
        }
    }

    return (
        <div className="make-post-page">
            <h1 className="make-post-heading">Add Post</h1>
            <form className="make-post-form" onSubmit={handleSubmit}>
                <textarea
                    className="make-post-textarea"
                    value={text}
                    onChange={(e) => setText(e.target.value)}
                    rows={8}
                    disabled={loading}
                />
                <div className="make-post-actions">
                    <button
                        type="submit"
                        className="btn btn--primary"
                        disabled={loading || text.trim() === ""}
                    >
                        {loading ? "Submitting…" : "Post"}
                    </button>
                </div>
            </form>
            {error && (
                <p className="app-alert make-post-alert" role="alert">
                    {error}
                </p>
            )}
        </div>
    )
}
