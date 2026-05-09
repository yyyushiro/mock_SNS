import { Navigate } from "react-router-dom"

/** Kept for bookmarks; same screen as timeline with public feed selected. */
export default function PublicFeedPage() {
    return <Navigate to="/timeline?view=feeds" replace />
}
