// =============================================================================
// page.go = 画面に渡す情報を組み立てる場所
//
//	Goのテンプレートは「渡したものしか見えない」。整理券もメッセージも、
//	渡さなければ画面に出せない。かといって全ハンドラに同じ数行を書くのは事故のもと。
//	そこで「毎回必要なもの」をここで自動的に詰める。
//
//	    return c.Render(200, "index.html", view.Page(c, view.Data{"Title": "ホーム"}))
//
//	検索結果とSNSでの見え方(Description / SiteURL / CanonicalURL)もここで詰めるので、
//	各ページのハンドラでは何も書かなくても最低限の形が出る。
//
// =============================================================================
package view

import (
	"github.com/labstack/echo/v5"

	"case_echo/internal/config"
	"case_echo/internal/middleware"
)

// Data = 画面に渡す情報の入れ物。「名前 → 中身」の辞書。
//
//	view.Data{"Title": "ホーム", "Posts": posts}
//
// ★左の名前が、HTML側の {{ .Title }} {{ .Posts }} と1対1で対応する。
type Data = map[string]any

// 設定は起動時に1回だけ受け取って覚えておく。
var appConfig *config.Config

// =============================================================================
// ★サイトの基本情報。ここ3つを直せば全ページに反映される。
// =============================================================================

// siteName = サイトの表示名。全ページのタイトルとヘッダーに出る。
const siteName = "Echo Template Demo"

// defaultDescription = 説明文を渡さなかったページで使われる文章。
//
// ★120文字前後に収めること。長いと検索結果で途中から切られる。
// ★ページごとに変えたいときは、ハンドラ側で "Description" を渡す。
const defaultDescription = "Echo(Go)のWebアプリ開発テンプレート。docker compose up だけで開発環境が立ち上がります。"

// ogpImage = SNSに貼ったときに出るサムネイル画像。
//
// ★空のときは画像用のタグを出さない。画像が無いのにタグだけ出すと、
//
//	SNS側が「画像が取れない」と判断してカードが小さいまま表示される。
//
// 用意するとき:
//
//	① 1200x630 の画像を web/static/images/ogp.png として置く
//	② ここを "/static/images/ogp.png" に書き換える
const ogpImage = ""

// Setup = テンプレートを準備して、設定を覚える。router.go から呼ばれる。
func Setup(cfg *config.Config, templateDir string) (*Renderer, error) {
	appConfig = cfg

	// 開発中だけ、HTMLを保存するたびに読み込み直す設定にする。
	return NewRenderer(templateDir, !cfg.IsProduction())
}

// Page = 全ページ共通の情報に、そのページ固有の情報を足して返す。
//
// 画面から使える共通の値:
//
//	{{ .SiteName }}     … サイトの表示名
//	{{ .Description }}  … 検索結果やSNSに出る説明文
//	{{ .SiteURL }}      … https://example.com(末尾スラッシュ無し)
//	{{ .CanonicalURL }} … 今開いているページの正式なURL
//	{{ .OGPImage }}     … SNSのサムネイル画像(未設定なら空)
//	{{ .AuthEnabled }}  … ログイン機能がONか
//	{{ .CurrentUser }}  … 今ログインしている人(未ログインなら空)
//	{{ .CSRFToken }}    … フォームに入れる整理券
//	{{ .Flashes }}      … 「保存しました」などのメッセージ
func Page(c *echo.Context, data Data) Data {
	if data == nil {
		data = Data{}
	}

	data["SiteName"] = siteName

	// ハンドラ側で渡していたらそちらを優先する。
	if _, ok := data["Description"]; !ok {
		data["Description"] = defaultDescription
	}

	base := siteURL(c)

	data["SiteURL"] = base

	// CanonicalURL = このページの正式なURL。
	// ★これが無いと ?utm_source=... が付いただけのURLが別ページとして数えられ、
	//   検索エンジンからの評価が分散する。
	data["CanonicalURL"] = base + c.Request().URL.Path

	data["OGPImage"] = ogpImage

	data["AuthEnabled"] = appConfig.AuthEnabled
	data["CurrentUser"] = middleware.CurrentUser(c)
	data["CSRFToken"] = middleware.CSRFToken(c)
	data["Flashes"] = middleware.TakeFlashes(c)

	return data
}

// siteURL = 「https://ドメイン」までを組み立てる。
//
// ▼ ★設定に書かず、アクセスされたURLから組み立てている理由
//
//	ドメインを設定に書くと、開発中(localhost)と本番で食い違い、
//	片方が必ず間違ったURLを出す。組み立てればどちらでも正しくなる。
//
// ▼ ★Nginxを通すと http に見える
//
//	本番では NginxとEchoの間はHTTPなので、そのままでは必ず http になる。
//	本物の入口がHTTPSだったかは X-Forwarded-Proto に入っていて
//	(付けているのは docker/nginx/app.inc)、c.Scheme() がそれを見てくれる。
//	★信じるのは、内輪のネットワーク(Dockerの中など)から届いたときだけ。
//	  インターネットから直接届いたヘッダーは無視されるので、詐称はできない。
func siteURL(c *echo.Context) string {
	return c.Scheme() + "://" + c.Request().Host
}
