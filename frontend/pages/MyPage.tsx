import { useEffect, useState, type FormEvent } from "react"
import { Link, useNavigate } from "react-router-dom"
import {
    deleteMyAccount,
    getMyInfo,
    logout,
    updateMyUsername,
    type MyUserInfo,
} from "../apis/API.ts"

export default function MyPage() {
    const navigate = useNavigate()
    const [info, setInfo] = useState<MyUserInfo | null>(null)
    const [username, setUsername] = useState("")
    const [loading, setLoading] = useState(true)
    const [loadError, setLoadError] = useState<string | null>(null)
    const [saving, setSaving] = useState(false)
    const [saveError, setSaveError] = useState<string | null>(null)
    const [loggingOut, setLoggingOut] = useState(false)
    const [logoutError, setLogoutError] = useState<string | null>(null)
    const [deleting, setDeleting] = useState(false)
    const [deleteError, setDeleteError] = useState<string | null>(null)

    useEffect(() => {
        let cancelled = false
        getMyInfo()
            .then((data) => {
                if (!cancelled) {
                    setInfo(data)
                    setUsername(data.username)
                }
            })
            .catch((err: unknown) => {
                if (!cancelled) {
                    setLoadError(
                        err instanceof Error ? err.message : "Failed to load",
                    )
                }
            })
            .finally(() => {
                if (!cancelled) setLoading(false)
            })
        return () => {
            cancelled = true
        }
    }, [])

    async function handleSaveUsername(e: FormEvent) {
        e.preventDefault()
        setSaveError(null)
        const next = username.trim()
        if (next === "") {
            setSaveError("Username cannot be empty.")
            return
        }
        setSaving(true)
        try {
            const updated = await updateMyUsername(next)
            setInfo(updated)
            setUsername(updated.username)
        } catch (err: unknown) {
            const msg = err instanceof Error ? err.message : "Could not save"
            if (
                err instanceof Error &&
                err.message.includes("Response status: 409")
            ) {
                setSaveError("That username is already taken.")
            } else if (
                err instanceof Error &&
                err.message.includes("Response status: 400")
            ) {
                setSaveError("Invalid username.")
            } else setSaveError(msg)
        } finally {
            setSaving(false)
        }
    }

    async function handleLogout() {
        setLogoutError(null)
        setLoggingOut(true)
        try {
            await logout()
            navigate("/", { replace: true })
        } catch (err: unknown) {
            setLogoutError(
                err instanceof Error ? err.message : "Could not log out",
            )
        } finally {
            setLoggingOut(false)
        }
    }

    async function handleDeleteAccount() {
        if (!window.confirm("アカウントを削除しますか？この操作は取り消せません。")) return
        setDeleteError(null)
        setDeleting(true)
        try {
            await deleteMyAccount()
            navigate("/", { replace: true })
        } catch (err: unknown) {
            setDeleteError(
                err instanceof Error ? err.message : "Could not delete account",
            )
        } finally {
            setDeleting(false)
        }
    }

    const displayHintId = info?.id ?? ""

    return (
        <div className="my-page">
            <header className="my-page__header">
                <Link to="/timeline" className="btn btn--primary">
                    Back to timeline
                </Link>
                <h1 className="my-page__title">My page</h1>
            </header>

            <main className="my-page__main">
                {loading && <p>Loading…</p>}
                {loadError && (
                    <p className="app-alert" role="alert">
                        {loadError}
                    </p>
                )}
                {!loading && !loadError && info && (
                    <>
                        <p className="my-page__meta">
                            User id (shown as placeholder when username is not
                            set):{" "}
                            <code className="my-page__code">{displayHintId}</code>
                        </p>

                        <form
                            className="my-page__form"
                            onSubmit={(e) => {
                                void handleSaveUsername(e)
                            }}
                        >
                            <label className="my-page__label" htmlFor="username">
                                Username
                            </label>
                            <input
                                id="username"
                                className="my-page__input"
                                type="text"
                                autoComplete="username"
                                maxLength={255}
                                value={username}
                                onChange={(e) => setUsername(e.target.value)}
                            />
                            {saveError && (
                                <p className="app-alert" role="alert">
                                    {saveError}
                                </p>
                            )}
                            <button
                                type="submit"
                                className="btn btn--primary"
                                disabled={saving}
                            >
                                {saving ? "Saving…" : "Save username"}
                            </button>
                        </form>

                        <section className="my-page__logout">
                            {logoutError && (
                                <p className="app-alert" role="alert">
                                    {logoutError}
                                </p>
                            )}
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
                        </section>

                        <section className="my-page__delete">
                            {deleteError && (
                                <p className="app-alert" role="alert">
                                    {deleteError}
                                </p>
                            )}
                            <button
                                type="button"
                                className="btn btn--danger"
                                disabled={deleting}
                                onClick={() => {
                                    void handleDeleteAccount()
                                }}
                            >
                                {deleting ? "Deleting…" : "Delete account"}
                            </button>
                        </section>
                    </>
                )}
            </main>
        </div>
    )
}
