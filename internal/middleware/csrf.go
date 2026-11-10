// =============================================================================
// csrf.go = 成りすまし送信を防ぐ仕掛け
//
// ▼ 何を防いでいるのか
//
//	うちのサイトにログインしたまま、別のサイトの罠を踏んだとする。
//	罠のページに「うちのサイトへ送信するフォーム」が仕込まれていると、
//	ブラウザはログインのメモを一緒に送ってしまうので、
//	本人が押したことになって勝手に投稿・削除されてしまう。
//
// ▼ 対策
//
//	フォームに「整理券」を埋め込み、整理券が無い依頼は受け付けない。
//	整理券はうちのページを開かないと手に入らないので、罠のページからは作れない。
//
// ▼ ★Echoには最初から入っている
//
//	確認の本体は Echo 付属の仕掛け。ここでは設定を渡しているだけ。
//	新しいブラウザが相手のときは、整理券より先に「この送信はどのサイトから
//	出たか」というブラウザ自身の申告(Sec-Fetch-Site)で判定している。
//	★そのときHTMLの整理券には _echo_csrf_using_sec_fetch_site_ という
//	  決まった文字が入る。「申告で判定済み」の印で、壊れているわけではない。
//
// ▼ 開発側でやること
//
//	フォーム …… <input type="hidden" name="csrf_token" value="{{ .CSRFToken }}">
//	JavaScript … main.js の api.post() を使う(整理券が自動で付く)
//
// =============================================================================
package middleware

import (
	"net/http"

	"github.com/labstack/echo/v5"
	echomw "github.com/labstack/echo/v5/middleware"
)

// 整理券を、1回のやり取りの間だけ持ち歩くときの名前。
const keyCSRFToken = "csrf_token"

// CSRF = 整理券を確認する仕掛け。router.go で全ページに取り付ける。
//
// ★echomw という名前で読み込んでいる理由
//
//	Echo付属の仕掛けも middleware という名前で、このフォルダと同じになる。
//	同じ名前は2つ同時に使えないので、借り物のほうに別名を付けている。
func CSRF(secure bool) echo.MiddlewareFunc {
	return echomw.CSRFWithConfig(echomw.CSRFConfig{
		Skipper: IsStaticPath,

		// 整理券を探す場所。JavaScriptからはヘッダーで、フォームからは隠し項目で届く。
		// ★名前を変えるなら、main.js とフォームの name も一緒に変えること。
		TokenLookup: "header:X-CSRF-Token,form:csrf_token",

		ContextKey: keyCSRFToken,

		// 整理券の控えはブラウザのメモ(Cookie)に置かれる。
		CookieName: "hackapp_csrf",

		// ★"/" を必ず書く。書かないと「開いたページのフォルダ専用」のメモになり、
		//   /auth/login で貰った整理券が /api/ への送信では使えなくなる。
		CookiePath: "/",

		// ログインのメモと同じ7日間。短いと、開きっぱなしの画面から送ったときに弾かれる。
		CookieMaxAge: 60 * 60 * 24 * 7,

		CookieHTTPOnly: true,
		CookieSecure:   secure,
		CookieSameSite: http.SameSiteLaxMode,

		ErrorHandler: rejectCSRF,
	})
}

// CSRFToken = この人の整理券を返す。
//
// 画面を作るとき(view.Page)から自動で呼ばれるので、自分で呼ぶことはない。
func CSRFToken(c *echo.Context) string {
	token, _ := c.Get(keyCSRFToken).(string)
	return token
}

// rejectCSRF = 整理券が無い/違うときの返事。
func rejectCSRF(c *echo.Context, _ error) error {
	// APIならJSONで返す。HTMLを返すとJavaScript側が読めずに混乱する。
	if IsAPIPath(c) {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "整理券(CSRFトークン)が正しくありません。ページを再読み込みしてください。",
		})
	}

	return c.String(http.StatusBadRequest,
		"整理券(CSRFトークン)が正しくありません。\n"+
			"フォームに次の1行が入っているか確認してください:\n"+
			`<input type="hidden" name="csrf_token" value="{{ .CSRFToken }}">`)
}
