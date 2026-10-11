import { LoaderCircle } from "lucide-preact";
import { render } from "preact";
import { Library } from "./components/Library";
import { Login } from "./components/Login";
import { useStatus } from "./hooks/useStatus";
import "./style.css";
function App() {
  const { status, error, refresh } = useStatus();
  if (!status)
    return (
      <div class="loading-screen">
        <img src="/icon-256.png" alt="" />
        <LoaderCircle class="spin" />
        <p>{error || "ArrScope wird geladen …"}</p>
      </div>
    );
  if (!status.authenticated) return <Login status={status} done={refresh} />;
  return <Library status={status} refresh={refresh} />;
}
render(<App />, document.getElementById("app")!);
