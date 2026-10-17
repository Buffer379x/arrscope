import { LoaderCircle } from "lucide-preact";
import { render } from "preact";
import { lazy, Suspense } from "preact/compat";
import { ConfirmationDialog } from "./components/ConfirmationDialog";
import { Login } from "./components/Login";
import { Notifications } from "./components/Notifications";
import { useStatus } from "./hooks/useStatus";
import { I18nProvider, t, useI18n } from "./i18n";
import "./style.css";
const Library = lazy(() =>
  import("./components/Library").then((module) => ({
    default: module.Library,
  })),
);
function App() {
  useI18n();
  const { status, error, refresh } = useStatus();
  const loading = (
    <div class="loading-screen">
      <img src="/icon-256.png" alt="" />
      <LoaderCircle class="spin" />
      <p>{error || t("Loading Scopearr …")}</p>
    </div>
  );
  if (!status) return loading;
  if (!status.authenticated) return <Login status={status} done={refresh} />;
  return (
    <Suspense fallback={loading}>
      <Library status={status} refresh={refresh} />
    </Suspense>
  );
}
render(
  <I18nProvider>
    <Notifications />
    <ConfirmationDialog />
    <App />
  </I18nProvider>,
  document.getElementById("app")!,
);
