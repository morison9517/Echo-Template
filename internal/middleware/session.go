// =============================================================================
// middleware = 「全部の通り道に置いておく仕掛け」を集めた場所
//
// ▼ ミドルウェアとは
//
//	お店の入口に置く検温器のようなもの。
//	どのページに行くお客さんも必ずここを通るので、
//	「毎回やること」を1か所にまとめられる。
//
//	    ブラウザ → [セッション] → [ログイン確認] → [成りすまし対策] → 各ページ
//
//	これが無いと、全部のページの先頭に同じコードを書く羽目になる。
//
// ▼ ★Echoの仕掛けの形
//
//	「次の処理(next)を受け取って、新しい処理を返す関数」として書く。
//	next(c) を呼べば先へ進み、呼ばずに return すればそこで止まる。
//
// -----------------------------------------------------------------------------
// このファイル(session.go)= ブラウザに小さなメモを持たせる仕組み
//
//	HTTPは「1回のやり取りが終わると相手を忘れる」仕組みなので、
//	そのままだとログインしても次のページで他人になってしまう。
//	そこでブラウザに「あなたは3番の人」というメモ(Cookie)を持たせる。
//
//	★メモの中身は SECRET_KEY で署名されている。
//	  中身を書き換えると署名が合わなくなるので、
//	  「自分は1番(管理者)です」と偽装できない。
//
// =============================================================================
package middleware

import (
	"encoding/gob"
	"net/http"
	"strings"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo/v5"
)

const (
	sessionName = "hackapp_session" // ブラウザに保存されるメモの名前
	keyUserID   = "user_id"         // メモの中の「誰か」の欄
	keySession  = "session"         // 1回のやり取りの間、メモを持ち歩くときの名前
)

// FlashMessage = 次の画面に一度だけ出すメッセージ。
//
//	Category … "success" / "error" / "warning"(見た目の色に対応)
//	Message  … 表示する文章
type FlashMessage struct {
	Category string
	Message  string
}

// ★この1行が無いとメッセージの保存時にエラーになる。
//
//	メモには文字や数字しか入れられないので、
//	「FlashMessage という形のものも入れます」と先に登録しておく必要がある。
//	init() は「このファイルが読み込まれたとき自動で1回だけ動く」関数。
func init() {
	gob.Register(FlashMessage{})
}

// Session = メモを扱えるようにする仕掛け。router.go で一番最初に取り付ける。
func Session(secretKey string, secure bool) echo.MiddlewareFunc {
	// 割り印(SECRET_KEY)を使ってメモに署名する係。
	store := sessions.NewCookieStore([]byte(secretKey))

	store.Options = &sessions.Options{
		Path:   "/",
		MaxAge: 60 * 60 * 24 * 7, // 7日間。過ぎたら自動でログアウト

		// JavaScriptからメモを読めなくする。
		// 万一ページに悪意のあるスクリプトを埋め込まれても、盗まれない。
		HttpOnly: true,

		// HTTPSのときだけメモを送る。開発中(http)は false でないと動かないので、
		// 本番かどうかで切り替えている。
		Secure: secure,

		// 他サイトからの遷移でメモを送らない(成りすましの入口をふさぐ)。
		SameSite: http.SameSiteLaxMode,
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			if IsStaticPath(c) {
				return next(c)
			}

			// ★エラーを捨てている(_)のはわざと。
			//   署名が合わないメモ(SECRET_KEY を変えた後の古いメモなど)は、
			//   エラーと一緒に「まっさらなメモ」が返ってくる。未ログインとして続ければよい。
			s, _ := store.Get(c.Request(), sessionName)

			// c.Set = この1回のやり取りの間だけ、名前を付けて物を預けておく棚。
			c.Set(keySession, s)

			return next(c)
		}
	}
}

// IsStaticPath = CSSや画像など「ファイルを渡すだけ」のURLか。
//
// ★Echoは e.Use() の仕掛けを、登録の順番に関係なく全部のURLに適用する。
//
//	何もしないと画像1枚ごとにログイン確認(DBへの問い合わせ)が走るので、
//	各仕掛けの入口でここを見て素通りさせている。
func IsStaticPath(c *echo.Context) bool {
	path := c.Request().URL.Path

	return strings.HasPrefix(path, "/static/") ||
		strings.HasPrefix(path, "/media/") ||
		path == "/robots.txt" ||
		path == "/favicon.ico"
}

// IsAPIPath = JavaScript向けのURL(JSONで返事をする相手)か。
//
// ★エラーの返し方を決めるのに使う。JSONを待っている相手にHTMLを返すと、
//
//	JS側が「JSONとして読めない」と別のエラーになり、本当の原因が消える。
func IsAPIPath(c *echo.Context) bool {
	path := c.Request().URL.Path

	return strings.HasPrefix(path, "/api/") || strings.HasPrefix(path, "/__demo/api/")
}

// current = このやり取りのメモを取り出す。無ければ nil。
//
// ★nil になるのは、上の IsStaticPath で素通りしたURLのとき。
func current(c *echo.Context) *sessions.Session {
	s, _ := c.Get(keySession).(*sessions.Session)
	return s
}

// save = メモをブラウザに送り返す。
//
// ★これを呼ばないとブラウザに届かない。
//
//	「ログインしたのに次のページで未ログインに戻る」の原因はほぼこれ。
func save(c *echo.Context, s *sessions.Session) error {
	return s.Save(c.Request(), c.Response())
}

// Login = 「あなたは○番の人」とメモに書き込む。
func Login(c *echo.Context, userID uint) error {
	s := current(c)
	if s == nil {
		return nil
	}

	s.Values[keyUserID] = userID
	return save(c, s)
}

// Logout = メモを丸ごと捨てる。
func Logout(c *echo.Context) error {
	s := current(c)
	if s == nil {
		return nil
	}

	// 1項目だけ消すのではなく全部消す理由:
	// 前の人の情報が1つでも残っていると事故のもとになるため。
	clear(s.Values)
	return save(c, s)
}

// UserID = メモに書かれている番号を読む。ログインしていなければ 0。
func UserID(c *echo.Context) uint {
	s := current(c)
	if s == nil {
		return 0
	}

	// ★.(uint) は「中身が uint 型なら取り出す」という書き方。
	//   メモには何でも入れられる形で保存されているので、取り出すときに型を指定する。
	if id, ok := s.Values[keyUserID].(uint); ok {
		return id
	}
	return 0
}

// -----------------------------------------------------------------------------
// flash = 次の画面に一度だけ出すメッセージ
//
//	表示する場所は base.html に1か所だけあるので、各ページで自作しなくてよい。
//	★「登録処理 → トップページへ移動 → そこで表示」のように、
//	  移動先の画面に持ち越せるのがポイント。
// -----------------------------------------------------------------------------

// Flash = メッセージを積んでおく。表示は次の画面で自動的に行われる。
//
//	middleware.Flash(c, "success", "保存しました")
func Flash(c *echo.Context, category, message string) {
	s := current(c)
	if s == nil {
		return
	}

	s.AddFlash(FlashMessage{Category: category, Message: message})
	_ = save(c, s)
}

// TakeFlashes = 溜まっているメッセージを取り出す(取り出したら消える)。
//
// 画面を作るとき(view.Page)から自動で呼ばれるので、自分で呼ぶことはない。
func TakeFlashes(c *echo.Context) []FlashMessage {
	s := current(c)
	if s == nil {
		return nil
	}

	raw := s.Flashes()
	if len(raw) == 0 {
		return nil
	}

	// ★取り出しただけでは消えない。save して初めてメモから消える。
	//   これを忘れると同じメッセージが毎ページ出続ける。
	_ = save(c, s)

	messages := make([]FlashMessage, 0, len(raw))
	for _, item := range raw {
		if m, ok := item.(FlashMessage); ok {
			messages = append(messages, m)
		}
	}
	return messages
}
