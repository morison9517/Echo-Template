// =============================================================================
// page.go = 画面(HTML)を返す受付
//
//	ブラウザ「/ をください」 → ここ → 「index.html をどうぞ」
//
// ▼ ★担当ごとにファイルを分けている
//
//	URLの登録を1つのファイルに全員が書くと、git で必ず衝突する。
//	自分のファイルの中で自分のURLを登録する形にしてある。
//
//	    page.go → 画面を返すURL      ← このファイル
//	    auth.go → ログイン関係のURL
//	    api.go  → JavaScript向けのURL
//
//	router.go はそれぞれの登録係を呼ぶだけ。普段は触らない。
//
// ▼ ★Echoのハンドラは必ず error を返す
//
//	return c.Render(...) のように、返事を作る行に return を付ける。
//	付け忘れるとビルドが通らないので、書き忘れには必ず気づける。
//
// ▼ ★トップページはまだ空いている
//
//	今 "/" を開くとデモページが出るのは、ここに "/" が無いから。
//	下の見本のコメントを外せば自分たちの画面に入れ替わる。
//	デモを消す作業は要らない(仕組みは internal/demo/demo.go)。
//
// =============================================================================
package handlers

import (
	"net/http"

	"github.com/labstack/echo/v5"
)

// RegisterPageRoutes = このファイルが担当するURLを登録する。
//
// ★画面を1枚増やすときの手順
//  1. web/templates/pages/ にHTMLを1枚置く
//  2. ここに e.GET("/URL", 関数名) を1行足す
//  3. その関数を下に書く
func RegisterPageRoutes(e *echo.Echo) {
	// ★ここから書きはじめる(コメントを外して、下の index も有効にする)
	// e.GET("/", index)

	e.GET("/health", health)
}

// ★トップページの見本(コメントを外して使う)
//
//	view.Page(c, ...) が、ここに書いた情報 + 共通の情報 をまとめてくれる。
//	使うときは import に "case_echo/internal/view" を足すこと。
//
//	func index(c *echo.Context) error {
//		return c.Render(http.StatusOK, "index.html", view.Page(c, view.Data{
//			"Title": "ホーム",
//		}))
//	}

// health = 動作確認用。c.JSON に辞書を渡すと、Echoが自動でJSONにする。
//
// AWSやNginxが「アプリが生きているか」を確認しに来る先。
// 開発中も「画面が出ない…アプリ自体は動いてる?」の切り分けに使える。
func health(c *echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}
