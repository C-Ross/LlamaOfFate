import { StrictMode } from "react"
import { createRoot } from "react-dom/client"
import "./index.css"
import ChallengeDemo from "./ChallengeDemo"

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <ChallengeDemo />
  </StrictMode>,
)
