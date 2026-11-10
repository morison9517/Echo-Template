// =============================================================================
// auth.go = ログイン・ログアウト・新規登録
//
// ▼ Echo側の受け取り方
//
//	GET  = 画面を見に来た  → HTMLを返す
//	POST = 入力を送ってきた → 中身を照合する
//	同じURLでも、登録するときに GET用/POST用 で別々の関数を指定して分ける。
//
// ▼ 入力の取り出し方
//
//	c.FormValue("username") の "username" は
//	HTML側の <input name="username"> と1対1で対応する。
//	★ここがズレると値が届かない(つまずきの定番)。
//
// =============================================================================
package handlers

import (
	"fmt"
	"log"
	"math"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/labstack/echo/v5"
	"gorm.io/gorm"

	"case_echo/internal/database"
	"case_echo/internal/middleware"
	"case_echo/internal/models"
	"case_echo/internal/view"
)

// RegisterAuthRoutes = ログイン関係のURLを登録する。
//
// Group("/auth") を挟むので、下の "/login" は実際には "/auth/login" になる。
func RegisterAuthRoutes(e *echo.Echo) {
	auth := e.Group("/auth")

	auth.GET("/register", registerPage)
	auth.POST("/register", registerSubmit)

	auth.GET("/login", loginPage)
	auth.POST("/login", loginSubmit)

	// ★ログアウトが POST な理由
	//   GETでログアウトできると、他サイトに <img src="/auth/logout"> と
	//   書かれただけで勝手にログアウトさせられてしまう。
	//   「状態を変える操作はPOST」が基本ルール。
	auth.POST("/logout", logoutSubmit)
}

// -----------------------------------------------------------------------------
// 新規登録
// -----------------------------------------------------------------------------

func registerPage(c *echo.Context) error {
	// もうログインしている人が登録画面に来たら、トップへ返す。
	if middleware.CurrentUser(c) != nil {
		return c.Redirect(http.StatusFound, "/")
	}

	return c.Render(http.StatusOK, "register.html", view.Page(c, view.Data{
		"Title": "新規登録",
	}))
}

func registerSubmit(c *echo.Context) error {
	if middleware.CurrentUser(c) != nil {
		return c.Redirect(http.StatusFound, "/")
	}

	username := strings.TrimSpace(c.FormValue("username"))
	password := c.FormValue("password")
	passwordConfirm := c.FormValue("password_confirm")

	// 問題点をまとめて集めて最後に表示する。
	// 「1個直すと次のエラーが出る」を繰り返させないため。
	var problems []string

	switch {
	case username == "":
		problems = append(problems, "ユーザー名を入力してください。")
	// ★len() は「バイト数」を数えるので、日本語だと1文字=3バイトになる。
	//   文字数を数えたいときは []rune に変換してから len を使う。
	case len([]rune(username)) > 80:
		problems = append(problems, "ユーザー名は80文字以内で入力してください。")
	}

	if len(password) < 8 {
		problems = append(problems, "パスワードは8文字以上で入力してください。")
	}

	if password != passwordConfirm {
		problems = append(problems, "パスワードが一致しません。")
	}

	if username != "" && usernameExists(username) {
		problems = append(problems, "そのユーザー名はすでに使われています。")
	}

	if len(problems) > 0 {
		for _, message := range problems {
			middleware.Flash(c, "error", message)
		}

		// username を渡すので、入力した名前は消えずに残る。
		return c.Render(http.StatusOK, "register.html", view.Page(c, view.Data{
			"Title":    "新規登録",
			"Username": username,
		}))
	}

	user := models.User{Username: username}
	if err := user.SetPassword(password); err != nil {
		middleware.Flash(c, "error", "登録に失敗しました。もう一度お試しください。")
		return c.Redirect(http.StatusFound, "/auth/register")
	}

	// ★ここで初めてDBに書き込まれる。
	//   &user の & は「コピーではなく本人を渡す」という意味。
	//   これが無いと、DBが振った番号(ID)が手元の user に反映されない。
	if err := database.DB.Create(&user).Error; err != nil {
		middleware.Flash(c, "error", "登録に失敗しました。もう一度お試しください。")
		return c.Redirect(http.StatusFound, "/auth/register")
	}

	_ = middleware.Login(c, user.ID)
	middleware.Flash(c, "success", "ようこそ、"+user.Username+" さん!")
	return c.Redirect(http.StatusFound, "/")
}

func usernameExists(username string) bool {
	var count int64
	database.DB.Model(&models.User{}).Where("username = ?", username).Count(&count)
	return count > 0
}

// -----------------------------------------------------------------------------
// ログイン
// -----------------------------------------------------------------------------

func loginPage(c *echo.Context) error {
	if middleware.CurrentUser(c) != nil {
		return c.Redirect(http.StatusFound, "/")
	}

	return c.Render(http.StatusOK, "login.html", view.Page(c, view.Data{
		"Title": "ログイン",
		// ログイン必須ページから飛ばされて来た場合、戻り先を持ち回す。
		"Next": c.QueryParam("next"),
	}))
}

func loginSubmit(c *echo.Context) error {
	if middleware.CurrentUser(c) != nil {
		return c.Redirect(http.StatusFound, "/")
	}

	username := strings.TrimSpace(c.FormValue("username"))
	password := c.FormValue("password")

	// ▼ ★何度も失敗している相手は、照合する前に追い返す(仕組みは loginlimit.go)
	//
	//   パスワードを見る前に止めるのが要点。照合の後ろに置くと、締め出し中でも
	//   1回ずつ試させてしまい、時間をかければ当てられる状態のままになる。
	ip := c.RealIP()

	if remaining := loginLockRemaining(ip); remaining > 0 {
		middleware.Flash(c, "error", fmt.Sprintf(
			"ログインの失敗が続いたため、一時的に制限しています。あと約%d分お待ちください。",
			minutesToWait(remaining)))

		return c.Render(http.StatusTooManyRequests, "login.html", view.Page(c, view.Data{
			"Title":    "ログイン",
			"Username": username,
			"Next":     c.FormValue("next"),
		}))
	}

	var user models.User

	// Where(...).First(...) = 条件に合う最初の1件を取る。
	// ★見つからないときもエラーとして返ってくる(gorm.ErrRecordNotFound)。
	//   「エラー = 異常」ではないので、区別して扱う。
	err := database.DB.Where("username = ?", username).First(&user).Error

	// ★「ユーザー名が違います」と書かない理由
	//   どの名前が実在するかを攻撃者に教えてしまうため、あえてぼかす。
	if err != nil || !user.CheckPassword(password) {
		if err != nil && err != gorm.ErrRecordNotFound {
			// DB自体の異常はログに残す(ログインの失敗とは別問題なので)。
			log.Printf("[auth] ログイン時のDBエラー: %v", err)
		}

		// 失敗を1回数える。5回目でこのアクセス元は15分締め出される。
		recordLoginFailure(ip)

		message := "ユーザー名またはパスワードが正しくありません。"
		status := http.StatusOK

		// ★今の失敗で締め出しに達したかを、その場で確かめて伝える。
		//   これが無いと、次に押したときに初めて締め出しを知ることになり、
		//   利用者からは「急に入れなくなった」としか見えない。
		if remaining := loginLockRemaining(ip); remaining > 0 {
			message = fmt.Sprintf(
				"ログインの失敗が続いたため、一時的に制限しました。あと約%d分お待ちください。",
				minutesToWait(remaining))
			status = http.StatusTooManyRequests
		} else if left := remainingLoginAttempts(ip); left > 0 && left <= 2 {
			// ★残り回数を出す理由は loginlimit.go の remainingLoginAttempts に。
			//   残り2回以下になってから出す(最初から出すと不安にさせるだけ)。
			message += fmt.Sprintf(
				"(あと%d回失敗すると、しばらくログインできなくなります)", left)
		}

		middleware.Flash(c, "error", message)
		return c.Render(status, "login.html", view.Page(c, view.Data{
			"Title":    "ログイン",
			"Username": username,
			"Next":     c.FormValue("next"),
		}))
	}

	// ★成功したら失敗の記録を消す。
	//   これが無いと、正しく入れた後も前の失敗が残り続け、
	//   次に1回打ち間違えただけで締め出されます。
	clearLoginFailures(ip)

	_ = middleware.Login(c, user.ID)
	middleware.Flash(c, "success", "ログインしました。")

	return c.Redirect(http.StatusFound, safeRedirect(c.FormValue("next")))
}

// minutesToWait = 残り時間を「あと約◯分」に直す。
//
// ★切り上げること。切り捨てると残り30秒で「あと約0分」と出て、
//
//	待っても入れないので壊れているように見える。必ず1分以上を返す。
func minutesToWait(remaining time.Duration) int {
	minutes := int(math.Ceil(remaining.Minutes()))
	if minutes < 1 {
		return 1
	}
	return minutes
}

// safeRedirect = ログイン後の遷移先が安全か確かめる。
//
// ログイン画面のURLには /auth/login?next=/mypage のように戻り先が付く。
// この next を無条件に信じると、?next=https://偽サイト.com というリンクを
// 配られてログイン直後に飛ばされる。自サイト内("/"始まり)だけを許可する。
func safeRedirect(target string) string {
	const fallback = "/"

	if target == "" {
		return fallback
	}

	parsed, err := url.Parse(target)
	if err != nil {
		return fallback
	}

	// 自サイト内なら /mypage の形なので、ドメイン名(Host)は空になる。
	if parsed.Scheme != "" || parsed.Host != "" {
		return fallback
	}

	// "//evil.com" は一見内部リンクに見えるが、外部サイトに飛ぶ書き方なので弾く。
	if !strings.HasPrefix(target, "/") || strings.HasPrefix(target, "//") {
		return fallback
	}

	// ★バックスラッシュ入りも弾く。ブラウザは \ を / と同じに扱うので、
	//   "/\evil.com" は上の確認をすり抜けたうえで "//evil.com" として外部へ飛ぶ。
	//   自サイトのURLに \ が入ることは無いので、含まれていたら丸ごと断る。
	if strings.Contains(target, `\`) {
		return fallback
	}

	return target
}

// -----------------------------------------------------------------------------
// ログアウト
// -----------------------------------------------------------------------------

func logoutSubmit(c *echo.Context) error {
	_ = middleware.Logout(c)

	// ★Logout でメモを全部消しているので、
	//   ここで先にメッセージを積んでも一緒に消えてしまう。順番が大事。
	middleware.Flash(c, "success", "ログアウトしました。")

	return c.Redirect(http.StatusFound, "/")
}
