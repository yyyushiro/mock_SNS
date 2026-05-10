import { Navigate } from "react-router-dom"

/** Bookmark-friendly URL; same screen as timeline with following feed selected. */
export default function FollowingFeedPage() {
    return <Navigate to="/timeline?view=following" replace />
}
