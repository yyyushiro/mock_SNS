import { useState } from "react"
import { useLocation, useNavigate } from "react-router-dom"
import { resendVerification } from "../apis/API"

type LocationState = { email?: string }

export default function VerifyEmailNoticePage() {
    const location = useLocation()
    const navigate = useNavigate()
    const state = location.state as LocationState | null
    const email = state?.email ?? ""

    const [resent, setResent] = useState(false)
    const [sending, setSending] = useState(false)

    if (!email) {
        navigate("/", { replace: true })
        return null
    }

    const handleResend = async () => {
        setSending(true)
        await resendVerification(email)
        setSending(false)
        setResent(true)
    }

    return (
        <div className="title-page">
            <h1 className="title-page__heading">メールを確認してください</h1>
            <p>{email} に確認メールを送信しました。</p>
            <p>メール内のリンクをクリックしてアカウントを有効化してください。</p>

            {resent ? (
                <p className="title-page__notice">再送しました。メールをご確認ください。</p>
            ) : (
                <button
                    type="button"
                    disabled={sending}
                    className="btn btn--secondary"
                    onClick={handleResend}
                >
                    {sending ? "送信中..." : "確認メールを再送する"}
                </button>
            )}
        </div>
    )
}
