// =============================================================================
// internal/demo/ = 動作確認用のデモページ一式(プロダクトには含まれません)
//
// ▼ このフォルダの決まり
//
//	・開発モードのときだけ取り付ける。本番には出ず、デモ用の表も作られない。
//	・トップページ("/")は handlers/page.go に "/" が無いときだけ引き受ける。
//	  自分たちのトップページを書いた瞬間に自動で出なくなる(消す作業は不要)。
//	・いつでも /__demo で開ける。
//	・★画面(page.html)は1枚完結。base.html も style.css も使わない。
//	  「セットアップが動いているか」を見せる計器なので、共通のファイルを
//	  作り替えても壊れないようにしてある。書き方の見本にはしないこと
//	  (見本は web/templates/pages/login.html)。
//
// ▼ ★Echoだけの注意点
//
//	Echoは同じURLを2回登録すると、エラーも出さずに「後から登録したほう」で
//	上書きする(Ginは起動時に落ち、FlaskとDjangoは先に登録したほうが勝つ)。
//	デモは最後に取り付けるので、何もしないとデモが勝ってしまう。
//	そこで下の hasRoute() で「"/" が空いているか」を先に確認している。
//	★この確認を消すと、自分たちのトップページを作ってもデモが出続ける。
//
// ▼ 不要になったら、このフォルダと router.go の「デモ」の数行を消すだけ。
//
// =============================================================================
package demo

import (
	"bytes"
	_ "embed"
	"html/template"
	"net/http"

	"github.com/labstack/echo/v5"

	"case_echo/internal/config"
	"case_echo/internal/database"
	"case_echo/internal/middleware"
)

// ★//go:embed = このHTMLをアプリ本体の中に取り込む指示。
//
//	web/templates/ に置くと共通の仕組みが自動で base.html と合体させてしまう。
//	1枚完結にしたいので、このフォルダに置いて自分で読み込んでいる。
//
//go:embed page.html
var pageHTML string

// デモ画面のテンプレート。起動時に1回だけ組み立てる。
var pageTmpl *template.Template

// 設定は Register() で受け取って覚えておく(画面に「ログイン機能:有効」を出すため)。
var appConfig *config.Config

// Register = デモを取り付ける。router.go から、開発モードのときだけ呼ばれる。
func Register(e *echo.Echo, cfg *config.Config) error {
	appConfig = cfg

	// デモ用の表を用意する。
	// ★ここでやっているので、本番のDBには todos 表が作られない。
	if err := database.DB.AutoMigrate(&Todo{}); err != nil {
		return err
	}

	// page.html を読み込む。書き間違いがあればここで気づける。
	tmpl, err := template.New("demo").Parse(pageHTML)
	if err != nil {
		return err
	}
	pageTmpl = tmpl

	// /__demo は常に開けるようにしておく(自分たちのトップページを作った後も
	// 「DBに繋がっているか」をここで確認できる)。
	e.GET("/__demo", index)
	registerAPIRoutes(e)

	// ★トップページが空いているときだけ、デモが "/" を引き受ける。
	//   handlers/page.go に "/" を書いた後は、この条件が外れてデモは出なくなる。
	//   (確認せずに登録すると、自分たちのトップページをデモが黙って上書きする)
	if !hasRoute(e, http.MethodGet, "/") {
		e.GET("/", index)
	}

	return nil
}

// hasRoute = そのURLが既に登録されているか調べる。
//
// e.Router().Routes() は「今この受付に登録されている全URLの一覧」。
// FindByMethodPath は、見つからなかったときにエラーを返す。
func hasRoute(e *echo.Echo, method, path string) bool {
	_, err := e.Router().Routes().FindByMethodPath(method, path)
	return err == nil
}

// index = デモページ。"/" と "/__demo" の両方から使われる。
//
// ★普通のページと違い、c.Render(...) を使っていない。
//
//	c.Render は base.html と合体させる共通の仕組みを通るので、
//	1枚完結のこのページには使えない。ここでは自分で組み立てて返している。
//	自分たちのページを作るときは c.Render を使うこと(見本は handlers/page.go)。
func index(c *echo.Context) error {
	dbStatus, dbMessage := checkDB()

	data := map[string]any{
		"Title":       "セットアップ確認",
		"DBStatus":    dbStatus,
		"DBMessage":   dbMessage,
		"AuthEnabled": appConfig.AuthEnabled,
		"CSRFToken":   middleware.CSRFToken(c),

		// ★真偽値だけを渡している(利用者オブジェクトそのものは渡さない)。
		//   画面に名前を出すと .Username を読むことになり、開発する人が
		//   Userの項目名を変えた日にデモが壊れる。
		//   「ログイン中か」だけならフレームワーク側の概念なので絶対に壊れない。
		"LoggedIn": middleware.CurrentUser(c) != nil,
	}

	// ★いったん下書き用紙(buf)に書いてから送る。
	//   直接ブラウザへ書くと、途中で失敗したときに書きかけの画面が出てしまう。
	var buf bytes.Buffer
	if err := pageTmpl.Execute(&buf, data); err != nil {
		return err
	}

	return c.HTMLBlob(http.StatusOK, buf.Bytes())
}

// checkDB = DBに繋がるか実際に試す。
//
// 「返事してください」という最小の確認を送り、返事が来るかで判定している。
//
// エラーで落とさず画面は出す理由:
// DBが起動しきっていないだけでこの画面が真っ白になると原因が分かりにくい。
// 画面は出しつつ「DBだけ未接続」と伝えたほうが切り分けが速い。
func checkDB() (string, string) {
	sqlDB, err := database.DB.DB()
	if err != nil {
		return "ng", truncate(err.Error())
	}

	if err := sqlDB.Ping(); err != nil {
		return "ng", truncate(err.Error())
	}

	return "ok", ""
}

// truncate = エラー文が数百文字になることがあるので、先頭だけにする。
func truncate(s string) string {
	const limit = 120
	if len(s) <= limit {
		return s
	}
	return s[:limit] + "..."
}
