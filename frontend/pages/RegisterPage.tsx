import { useState } from "react"
import { useNavigate } from "react-router-dom"
import { register, RegisterError } from "../apis/API"

export default function RegisterPage() {
    const navigate = useNavigate()
    const [email, setEmail] = useState("")
    const [password, setPassword] = useState("")
    const [error, setError] = useState<string | null>(null)
    const [submitting, setSubmitting] = useState(false)

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault()
        setError(null)
        setSubmitting(true)
        try {
            await register(email, password)
            navigate("/verify-email-notice", { state: { email } })
        } catch (err) {
            if (err instanceof RegisterError) {
                setError("このメールアドレスはすでに登録されています")
            } else {
                setError("エラーが発生しました。もう一度お試しください。")
            }
        } finally {
            setSubmitting(false)
        }
    }

    return (
        <div className="title-page">
            <h1 className="title-page__heading">メールで新規登録</h1>

            <form onSubmit={handleSubmit} className="title-page__form">
                <input
                    type="email"
                    placeholder="メールアドレス"
                    value={email}
                    onChange={(e) => setEmail(e.target.value)}
                    required
                    className="title-page__input"
                />
                <input
                    type="password"
                    placeholder="パスワード"
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    required
                    className="title-page__input"
                />
                {error && <p className="title-page__error">{error}</p>}
                <button
                    type="submit"
                    disabled={submitting}
                    className="btn btn--primary"
                >
                    {submitting ? "登録中..." : "新規登録"}
                </button>
            </form>

            <button
                type="button"
                className="btn btn--secondary"
                onClick={() => navigate("/")}
            >
                戻る
            </button>
        </div>
    )
}
