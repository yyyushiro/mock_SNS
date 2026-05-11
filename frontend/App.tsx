import { BrowserRouter, Routes, Route } from "react-router-dom"
import TitlePage from './pages/TitlePage.tsx'
import TimeLinePage from "./pages/TimelinePage.tsx"
import MakePostPage from "./pages/MakePostPage.tsx"
import PublicFeedPage from "./pages/PublicFeedPage.tsx"
import FollowingFeedPage from "./pages/FollowingFeedPage.tsx"
import MyPage from "./pages/MyPage.tsx"

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path='/' element={<TitlePage />} />
        <Route path="/timeline" element={<TimeLinePage />} />
        <Route path="/mypage" element={<MyPage />} />
        <Route path="/explore" element={<PublicFeedPage />} />
        <Route path="/following" element={<FollowingFeedPage />} />
        <Route path="/post" element={<MakePostPage />} />
      </Routes>
    </BrowserRouter>
  )
}
