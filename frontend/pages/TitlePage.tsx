import { useNavigate } from "react-router-dom"

export default function TitlePage() {
    const navigate = useNavigate()

    return (
        <div className="title-page">
            <h1 className="title-page__heading">みんなの日記帳</h1>
            <button
                type="button"
                className="btn btn--primary"
                onClick={() => navigate("/register")}
            >
                メールで新規登録
            </button>
            <button
                type="button"
                className="btn btn--primary"
                onClick={() => navigate("/login")}
            >
                メールでログイン
            </button>
            <button
                type="button"
                className="btn btn--secondary"
                onClick={() => { window.location.href = "/api/auth/google/start" }}
            >
                Googleでログイン
            </button>
        </div>
    )
}
