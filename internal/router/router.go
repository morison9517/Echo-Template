// =============================================================================
// router.go = URLの受付をまとめて組み立てる場所
//
// ★このファイルは土台。普段は触らない。
//
//	URLを増やすときに触るのは handlers/ の中のファイル。
//	ここを触るのは「新しい担当ファイルを増やしたとき」だけ(下に1行足す)。
//
// ▼ 上から順に「通り道」を作っていくイメージ
//
//	ブラウザ
//	  ↓
//	[記録・保険]   アクセスを記録し、途中で落ちてもアプリ全体は止めない
//	  ↓
//	[セッション]   ブラウザのメモを読めるようにする
//	  ↓
//	[ログイン確認] 今アクセスしているのが誰か調べる
//	  ↓
//	[成りすまし対策] 送信に整理券が付いているか確認する
//	  ↓
//	[各ページ]     handlers/ の中の関数
//	  ↓
//	[受け皿]       エラーが返ってきたらエラー画面にする(handlers/error.go)
//
//	★CSS・JS・画像も同じ道を通るが、下3つの仕掛けは素通りする
//	  (middleware/session.go の IsStaticPath)。
//
// =============================================================================
package router

import (
	"fmt"
	"net"

	"github.com/labstack/echo/v5"
	echomw "github.com/labstack/echo/v5/middleware"

	"case_echo/internal/config"
	"case_echo/internal/demo"
	"case_echo/internal/handlers"
	"case_echo/internal/middleware"
	"case_echo/internal/view"
)

// ファイルの置き場所。プロジェクトの入口(compose.ymlのある場所)から見た位置。
const (
	templateDir = "web/templates"
	staticDir   = "web/static"
)

// アップロードの上限。
// ★本番のNginx側(docker/nginx/app.inc)とも同じ値にしておく。
const maxBodySize = 16 << 20 // 16MB

// New = アプリの受付を組み立てて返す。
func New(cfg *config.Config) (*echo.Echo, error) {
	e := echo.New()

	// ▼ ★Recover = 1か所でエラーが出ても、アプリ全体を止めない保険
	//
	//   これが無いと、誰か1人のページが落ちた瞬間にサーバーごと止まる。
	//   落ちた原因は 500 のエラーに変えられて handlers/error.go へ届く。
	//   ★AccessLog を先に書いているのは、落ちたアクセスも記録に残すため。
	e.Use(middleware.AccessLog(), echomw.Recover())

	// 上限を超えた送信は、中身を読み切る前に 413 で断る。
	e.Use(echomw.BodyLimit(maxBodySize))

	// ★「アクセス元のIPをどこまで信用するか」の設定。
	//
	//   Nginxを前に置くとEchoから見た相手はNginxになる。本当のアクセス元は
	//   ヘッダーに書いてあるが、無条件で信じると利用者が自分で詐称できる。
	//   開発中は空(誰も信用しない)。本番は .env の TRUSTED_PROXIES に書く。
	extractor, err := ipExtractor(cfg.TrustedProxies)
	if err != nil {
		return nil, err
	}
	e.IPExtractor = extractor

	// --- 画面の型紙(base.html)を使えるようにする ---
	renderer, err := view.Setup(cfg, templateDir)
	if err != nil {
		return nil, err
	}
	e.Renderer = renderer

	// --- CSS / JS / 画像 ---
	e.Static("/static", staticDir)

	// --- サイト直下に置かなければならない2つ ---
	// ★検索エンジンは /robots.txt しか見に来ない(/static/ に置いても読まれない)。
	// ★ブラウザは最後の手段として /favicon.ico を取りに来る。
	//   ファイルが無ければ404が返るだけで害はない。
	e.File("/robots.txt", staticDir+"/robots.txt")
	e.File("/favicon.ico", staticDir+"/favicon.ico")

	// --- 利用者が上げたファイル ---
	// ★開発モードのときだけEchoが配る。本番ではNginxが配る(compose.prod.yml)。
	if !cfg.IsProduction() {
		e.Static("/media", cfg.UploadDir)
	}

	// --- 全ページ共通の仕掛け(順番に意味がある) ---
	// ★secure が「HTTPSのときだけメモを送る」の指定。
	//   練習中はHTTPなので .env の SECURE_COOKIES で切り替えられる。
	secure := cfg.IsProduction() && cfg.SecureCookies

	e.Use(middleware.Session(cfg.SecretKey, secure))
	e.Use(middleware.LoadUser()) // セッションが読めないと誰か分からないので、後
	e.Use(middleware.CSRF(secure))

	// --- 担当ごとの受付を取り付ける ---
	// ★新しい担当ファイル(例:handlers/room.go)を作ったら、ここに1行足す。
	handlers.RegisterPageRoutes(e)
	handlers.RegisterAPIRoutes(e)

	// AUTH_ENABLED が false なら取り付けない(コードを消さずにOFFにできる)。
	if cfg.AuthEnabled {
		handlers.RegisterAuthRoutes(e)
	}

	// --- デモ(動作確認用のページ) ---
	// ★必ず最後に取り付ける。「自分たちのトップページが既にあるか」を見てから
	//   動くので、先に取り付けると判定できず、デモがトップページを取ってしまう。
	if !cfg.IsProduction() {
		if err := demo.Register(e, cfg); err != nil {
			return nil, err
		}
	}

	// --- エラー画面(404など)の受け皿 ---
	handlers.RegisterErrorRoutes(e, !cfg.IsProduction())

	return e, nil
}

// ipExtractor = 「アクセス元のIPの調べ方」を、設定に合わせて選ぶ。
//
//	信用する相手が居ない … 直接つないできた相手のIPをそのまま使う
//	信用する相手が居る   … その相手から来たときだけ X-Forwarded-For を読む
//
// ★ログイン試行制限(handlers/loginlimit.go)はこのIPで数えている。
//
//	ここが甘いと、ヘッダーを1回ごとに書き換えるだけで制限をすり抜けられる。
func ipExtractor(trustedProxies []string) (echo.IPExtractor, error) {
	if len(trustedProxies) == 0 {
		return echo.ExtractIPDirect(), nil
	}

	// ★Echoは何も言わないと「内輪のアドレスは全部信用する」ので、まず全部切る。
	//   そのうえで、.env に書かれた範囲だけを信用する相手として足していく。
	options := []echo.TrustOption{
		echo.TrustLoopback(false),
		echo.TrustLinkLocal(false),
		echo.TrustPrivateNet(false),
	}

	for _, proxy := range trustedProxies {
		// "172.16.0.0/12" のような書き方を、Goが扱える「アドレスの範囲」に直す。
		_, ipRange, err := net.ParseCIDR(proxy)
		if err != nil {
			return nil, fmt.Errorf(
				"TRUSTED_PROXIES の書き方が正しくありません(例: 172.16.0.0/12): %q", proxy)
		}
		options = append(options, echo.TrustIPRange(ipRange))
	}

	return echo.ExtractIPFromXFFHeader(options...), nil
}
