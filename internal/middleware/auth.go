// =============================================================================
// auth.go = 「今アクセスしているのは誰か」を毎回調べる仕掛け
//
//	ブラウザのメモには「あなたは3番の人」としか書かれていない。
//	その番号から実際のユーザー情報をDBから取ってくるのがここ。
//
// =============================================================================
package middleware

import (
	"net/http"
	"net/url"

	"github.com/labstack/echo/v5"

	"case_echo/internal/database"
	"case_echo/internal/models"
)

// この名前で「今の人」を1回のやり取りの間だけ持ち歩く。
const keyCurrentUser = "currentUser"

// LoadUser = 毎回のアクセスで「今の人」を調べて持たせる。
//
// ★ログインしていない人も普通に通す。
//
//	ここは「調べるだけ」の係。通す/通さないの判断は RequireLogin の役目。
func LoadUser() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			id := UserID(c)
			if id == 0 {
				// メモが無い = 未ログイン。そのまま次へ。
				return next(c)
			}

			var user models.User
			if err := database.DB.First(&user, id).Error; err != nil {
				// 番号は書いてあるのにDBに居ない(退会した等)。
				// 古いメモを持たせたままだと毎回DBを探しに行くので捨てる。
				_ = Logout(c)
				return next(c)
			}

			c.Set(keyCurrentUser, &user)
			return next(c)
		}
	}
}

// CurrentUser = 今アクセスしている人を取り出す。未ログインなら nil。
//
//	user := middleware.CurrentUser(c)
//	if user != nil { ... }
func CurrentUser(c *echo.Context) *models.User {
	user, _ := c.Get(keyCurrentUser).(*models.User)
	return user
}

// RequireLogin = 「ログイン中の人だけ通す」ドア。画面を返すページ用。
//
// 会員専用ページを作りたいときは、ルートの登録の最後に足すだけ:
//
//	e.GET("/mypage", myPage, middleware.RequireLogin())
//
// ★Echoでは「関数名の後ろ」に書く(Ginは前)。前に書くとビルドが通らない。
func RequireLogin() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			if CurrentUser(c) != nil {
				return next(c)
			}

			Flash(c, "warning", "このページを見るにはログインが必要です。")

			// 元々見たかったページを ?next= に付けておくと、
			// ログイン後にそこへ戻してあげられる(処理は handlers/auth.go)。
			target := url.QueryEscape(c.Request().URL.RequestURI())

			// ★next(c) を呼ばずに return しているので、本来のページの処理は動かない。
			return c.Redirect(http.StatusFound, "/auth/login?next="+target)
		}
	}
}

// RequireLoginAPI = 同じく「ログイン中だけ通す」ドアだが、JSONを返すAPI用。
//
// APIでログイン画面にリダイレクトすると、JavaScript側がHTMLを受け取って
// 「JSONとして読めない」と混乱するので、素直に「401 = 未ログイン」を返す。
//
//	api.POST("/todos", createTodo, middleware.RequireLoginAPI())
func RequireLoginAPI() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			if CurrentUser(c) != nil {
				return next(c)
			}

			return c.JSON(http.StatusUnauthorized, map[string]string{
				"error": "ログインが必要です。",
			})
		}
	}
}
