import { Notifications } from "./components/Notifications";
import { I18nProvider, t, useI18n } from "./i18n";
import { LoaderCircle } from "lucide-preact";
import { render } from "preact";
import { Library } from "./components/Library";
import { Login } from "./components/Login";
import { useStatus } from "./hooks/useStatus";
import "./style.css";
function App() {
  useI18n();
  const { status, error, refresh } = useStatus();
  if (!status)
    return (
      <div class="loading-screen">
        <img src="/icon-256.png" alt="" />
        <LoaderCircle class="spin" />
        <p>{error || t("Loading Scopearr …")}</p>
      </div>
    );
  if (!status.authenticated) return <Login status={status} done={refresh} />;
  return <Library status={status} refresh={refresh} connectionError={error} />;
}
render(
  <I18nProvider>
    <Notifications />
    <App />
  </I18nProvider>,
  document.getElementById("app")!,
);
