// =============================================================================
// error.go = エラー画面(400 / 401 / 403 / 404 / 500 など)を返す担当
//
// ▼ 使い方
//
//	return echo.ErrNotFound      // 404
//	return echo.ErrForbidden     // 403
//
//	★Echoのハンドラは「エラーを return する」だけでよい。
//	  返されたエラーは全部このファイルに集まり、ここが画面にして返す。
//
// ▼ 自動で出る場合が2つある
//
//	① 存在しないURL … Echoが自分で 404 のエラーを作ってここへ渡す
//	② 処理の途中で落ちたとき … router.go の Recover が 500 のエラーに変えて渡す
//
// ▼ 文言を直す・種類を増やす
//
//	下の errorPages の表を書き換えるだけ。HTML(error.html)は共通の1枚。
//
// ▼ 消したいとき
//
//	このファイルと web/templates/pages/error.html を消し、router.go の
//	RegisterErrorRoutes を外す(EchoがJSONだけの素っ気ないエラーを返すようになる)。
//
// =============================================================================
package handlers

import (
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v5"

	"case_echo/internal/middleware"
	"case_echo/internal/view"
)

// errorPage = 画面に出す2行(英語の種類名と日本語の説明)。
type errorPage struct {
	Name    string
	Message string
}

// errorPages = 状態番号ごとの文言。★文言を直すのはここ。
//
// ★400番台は送ってきた側の問題、500番台はこちら側の問題。
//
//	この区切りは検索エンジンも見ている。「ページが無い」ときに200を返すと、
//	無いページが検索結果に載り続ける。
//
// ★401 と 403 の違いは「誰か分からない」と「見せられない」。
//
//	このテンプレートのログイン必須ページは、403ではなくログイン画面へ
//	案内している(middleware/auth.go)。そのほうが親切なため。
var errorPages = map[int]errorPage{
	http.StatusBadRequest: {
		Name:    "Bad Request",
		Message: "リクエストの内容に誤りがあります。",
	},
	http.StatusUnauthorized: {
		Name:    "Unauthorized",
		Message: "このページを表示する権限がありません。",
	},
	http.StatusForbidden: {
		Name:    "Forbidden",
		Message: "このページを表示する権限がありません。",
	},
	http.StatusNotFound: {
		Name:    "Not Found",
		Message: "指定されたページが見つかりませんでした。",
	},

	// ★下の2つはEchoが自分で出す番号。
	//   405 … URLはあるが、その方法(GET/POST)では受け付けていない
	//   413 … 送ってきたデータが router.go の上限(16MB)を超えた
	http.StatusMethodNotAllowed: {
		Name:    "Method Not Allowed",
		Message: "このURLでは、その操作を受け付けていません。",
	},
	http.StatusRequestEntityTooLarge: {
		Name:    "Payload Too Large",
		Message: "送信したデータが大きすぎます。",
	},

	http.StatusInternalServerError: {
		Name:    "Internal Server Error",
		Message: "サーバー側で問題が発生しました。時間をおいて試してください。",
	},
}

// fallbackErrorPage = 表に無い番号が来たときの文言。
//
// ★表に無いからといって何も返さないと真っ白な画面になる。
var fallbackErrorPage = errorPage{
	Name:    "Error",
	Message: "問題が発生しました。",
}

// RegisterErrorRoutes = エラーの集まる先を、このファイルに付け替える。
//
// e.HTTPErrorHandler = 「ハンドラや仕掛けが return したエラー」が最後に届く場所。
// 特定のURLではないので、他の登録と取り合いにならない。
//
// debug が true(開発中)のときだけ、テンプレートの書き間違いを画面にそのまま出す。
func RegisterErrorRoutes(e *echo.Echo, debug bool) {
	e.HTTPErrorHandler = func(c *echo.Context, err error) {
		// ★もう返事を送り始めていたら、何もしない。
		//   1回のやり取りで返事は1つだけ。2つ目を書こうとすると画面が壊れる。
		if res, _ := echo.UnwrapResponse(c.Response()); res != nil && res.Committed {
			return
		}

		// errors.As = 「このエラーは○○という種類か」を調べ、そうなら取り出す。
		var tmplErr *view.TemplateError
		if errors.As(err, &tmplErr) {
			log.Printf("[template] %v", tmplErr)

			// ★本番では出さない(ファイルの構成が外に漏れるため)。下の500の画面になる。
			if debug {
				_ = c.String(http.StatusInternalServerError,
					"テンプレートのエラー\n\n"+tmplErr.Error()+"\n")
				return
			}
		}

		// エラーの中に番号が入っていればそれを使う。無ければ 500(こちら側の問題)。
		status := echo.StatusCode(err)
		if status == 0 {
			status = http.StatusInternalServerError
		}

		// ★500番台だけ原因をログに残す(画面に出すとコードの中身が漏れる)。
		//   Echoは自分ではエラーをログに出さないので、ここで出さないと原因が消える。
		if status >= http.StatusInternalServerError {
			log.Printf("[error] %s %s: %v", c.Request().Method, c.Request().URL.Path, err)
		}

		showError(c, status)
	}
}

// showError = エラー画面を返す。
//
// ★JavaScript向けのURL(/api/...)にはHTMLではなくJSONを返す。
//
//	main.js は {"error": "..."} の形を待っているので、HTMLを返すと
//	JS側が「JSONとして読めない」と別のエラーになり、本当の原因が消える。
func showError(c *echo.Context, status int) {
	page, ok := errorPages[status]
	if !ok {
		page = fallbackErrorPage
	}

	if middleware.IsAPIPath(c) {
		_ = c.JSON(status, map[string]string{"error": page.Message})
		return
	}

	err := c.Render(status, "error.html", view.Page(c, view.Data{
		"Title": strconv.Itoa(status),

		"ErrorCode":    status,
		"ErrorName":    page.Name,
		"ErrorMessage": page.Message,
	}))

	// ★エラー画面そのものが壊れていたときの最後の砦。文字だけでも必ず返す。
	//   ここでまたエラーを返すと、エラーの処理がエラーになって何も出なくなる。
	if err != nil {
		log.Printf("[error] エラー画面を表示できませんでした: %v", err)
		_ = c.String(status, page.Message)
	}
}
