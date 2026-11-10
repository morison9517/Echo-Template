// =============================================================================
// accesslog.go = 「誰がどのURLを開いて、何番で返したか」を1行ずつログに出す仕掛け
//
//	[アクセス] 200 GET    /auth/login (1.2ms)
//
//	★Echo付属のもの(RequestLogger)もあるが、1件が長いJSONで出るので
//	  目で追いにくい。開発中に読みやすい形にするため自前で書いている。
//
// ▼ 消したいとき
//
//	このファイルを消し、router.go の middleware.AccessLog() を外す。
//
// =============================================================================
package middleware

import (
	"log"
	"time"

	"github.com/labstack/echo/v5"
)

// AccessLog = アクセスを1行ずつ記録する。router.go で最初に取り付ける。
func AccessLog() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			start := time.Now()

			// 先に本来の処理を済ませる。終わってからでないと結果の番号が分からない。
			err := next(c)

			// ★エラーで終わったときは、まだ返事を送っていない(送るのは handlers/error.go)。
			//   その場合もエラーの中身から「何番で返すことになるか」を割り出してくれる。
			_, status := echo.ResolveResponseStatus(c.Response(), err)

			req := c.Request()
			log.Printf("[アクセス] %d %-6s %s (%v)",
				status, req.Method, req.URL.RequestURI(), time.Since(start).Round(100*time.Microsecond))

			// ★err をそのまま返すこと。握りつぶすとエラー画面が出なくなる。
			return err
		}
	}
}
