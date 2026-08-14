import { BRAND_ASSETS } from "../ui/brand-assets";

export function LoginPage({
  message,
  onLogin,
}: {
  message?: string;
  onLogin(): void;
}) {
  return (
    <main className="role-select-page login-auth-page" data-testid="login-page">
      <button
        className="login-skip"
        onClick={() => document.querySelector<HTMLElement>("#login-authentication")?.focus()}
        type="button"
      >
        Skip to sign in
      </button>
      <section className="login-hero" aria-labelledby="login-hero-title">
        <img className="login-hero__texture" src={BRAND_ASSETS.loginTexture} alt="" aria-hidden="true" />
        <div className="login-hero__brand">
          <img className="login-hero__logo" src={BRAND_ASSETS.mark} alt="" aria-hidden="true" />
          <div>
            <div className="login__title" role="heading" aria-level={2}>AviaSurveil360</div>
            <div className="login__sub">Civil Aviation Authority surveillance &amp; oversight</div>
          </div>
        </div>
        <div className="login-hero__story">
          <span className="login-demo-badge">Secure oversight workspace</span>
          <h1 id="login-hero-title">
            From plan
            <br />
            to closure.
          </h1>
          <span className="login-hero__rule" aria-hidden="true" />
          <p className="login-auth-hero-copy">
            Review assigned surveillance work with clear authority, evidence, and accountability at every step.
          </p>
        </div>
        <div className="login-hero__foot">
          <p>Server-authorized access</p>
          <span className="login-auth-hero-mark">CAA oversight platform</span>
        </div>
      </section>
      <section className="login-selector login-auth-panel" id="login-authentication" tabIndex={-1} aria-labelledby="login-auth-title">
        <div className="login-auth-card">
          <span className="login-selector__eyebrow">Organization access</span>
          <h2 id="login-auth-title">Sign in to AviaSurveil360</h2>
          <p className="login-auth-lede">
            Use your organization identity to access assigned oversight work.
          </p>
          <div className="login-auth-status" role="status">
            <span className="login-auth-status__dot" aria-hidden="true" />
            <span>Protected by server-side session authorization</span>
          </div>
          {message ? <p className="command-error" role="alert">{message}</p> : null}
          <button className="primary-button login-auth-button" onClick={onLogin} type="button">
            Sign in with organization identity
            <span aria-hidden="true">→</span>
          </button>
          <p className="login-auth-note">
            You will be redirected to your organization identity provider and returned to this workspace after sign-in.
          </p>
        </div>
      </section>
    </main>
  );
}
