package handlers

import (
"fmt"
"html"

"github.com/gofiber/fiber/v2"
)

// PlaylistShareRedirect serves an HTML page that deep-links into the app.
// URL: GET /pl/:token
// Deep link: weworship://playlist/:token/join
func (h *Handler) PlaylistShareRedirect(c *fiber.Ctx) error {
token := html.EscapeString(c.Params("token"))
deepLink := fmt.Sprintf("weworship://playlist/%s/join", token)

page := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <title>Join Playlist – WeWorship</title>
  <meta property="og:title" content="You've been invited to a WeWorship playlist" />
  <meta property="og:description" content="Open WeWorship to join the shared playlist." />
  <style>
    *, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }
    body {
      background: #0a0505;
      color: #fff;
      font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
      min-height: 100dvh;
      display: flex;
      flex-direction: column;
      align-items: center;
      justify-content: center;
      padding: 24px;
      gap: 0;
    }
    .glow {
      position: fixed;
      border-radius: 50%%;
      filter: blur(100px);
      pointer-events: none;
    }
    .glow-1 { top: -10%%; left: -10%%; width: 50%%; height: 50%%; background: rgba(148,0,0,0.25); }
    .glow-2 { bottom: -10%%; right: -10%%; width: 50%%; height: 50%%; background: rgba(148,0,0,0.15); }
    .card {
      position: relative;
      background: #1a0b0b;
      border: 1px solid rgba(255,255,255,0.06);
      border-radius: 32px;
      padding: 40px 32px;
      width: 100%%;
      max-width: 380px;
      text-align: center;
    }
    .logo {
      width: 56px;
      height: 56px;
      background: #940000;
      border-radius: 50%%;
      display: flex;
      align-items: center;
      justify-content: center;
      margin: 0 auto 20px;
      font-size: 24px;
    }
    h1 {
      font-size: 22px;
      font-weight: 800;
      margin-bottom: 8px;
      letter-spacing: -0.3px;
    }
    p {
      color: rgba(255,255,255,0.45);
      font-size: 14px;
      line-height: 1.6;
      margin-bottom: 32px;
    }
    .btn {
      display: block;
      width: 100%%;
      padding: 16px;
      background: #940000;
      color: #fff;
      font-size: 16px;
      font-weight: 700;
      border: none;
      border-radius: 16px;
      cursor: pointer;
      text-decoration: none;
      margin-bottom: 12px;
      transition: background 0.15s;
    }
    .btn:active { background: #b00000; }
    .hint {
      font-size: 12px;
      color: rgba(255,255,255,0.25);
      margin-top: 16px;
    }
  </style>
</head>
<body>
  <div class="glow glow-1"></div>
  <div class="glow glow-2"></div>

  <div class="card">
    <div class="logo">🎵</div>
    <h1>Join this playlist</h1>
    <p>You've been invited to collaborate on a worship playlist in WeWorship.</p>

    <a class="btn" href="%s" id="deeplink">Open in WeWorship</a>

    <p class="hint">Don't have the app? Download WeWorship from the App Store or Play Store.</p>
  </div>

  <script>
    // Auto-redirect after a short delay
    setTimeout(function() {
      window.location.href = "%s";
    }, 800);
  </script>
</body>
</html>`, deepLink, deepLink)

c.Set("Content-Type", "text/html; charset=utf-8")
return c.SendString(page)
}
