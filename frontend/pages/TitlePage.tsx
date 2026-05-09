export default function TitlePage() {
    const handleGoogleLogin = () => {
        window.location.href = '/api/auth/google/start'
    }

    return (
        <div className="title-page">
            <h1 className="title-page__heading">みんなの日記帳</h1>
            <button
                type="button"
                className="btn btn--primary"
                onClick={handleGoogleLogin}
            >
                Googleでログイン
            </button>
        </div>
    )
}
