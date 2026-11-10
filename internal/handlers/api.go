// =============================================================================
// api.go = JavaScript向けの受付(画面ではなくデータを返す)
//
// ▼ page.go との使い分け
//
//	ページ移動を伴う操作(ログイン、詳細ページへ移動) → page.go
//	その場で追加・削除・チェック(いいね、Todo追加)   → api.go
//
//	送受信の作法(整理券を付ける、エラーを拾う)は main.js の api がやるので、
//	JS側は api.post("/api/○○", { ... }) と書くだけでよい。
//
// ▼ ★動く見本は internal/demo/api.go にある(一覧・追加・切り替え・削除の一式)
//
// =============================================================================
package handlers

import (
	"github.com/labstack/echo/v5"
)

// RegisterAPIRoutes = JavaScript向けのURLを登録する。
func RegisterAPIRoutes(e *echo.Echo) {
	// Group("/api") = 「この先のURLは全部 /api で始まる」というまとめ方。
	api := e.Group("/api")

	// ★ログイン必須にしたい場合は、この1行を足す:
	//     api.Use(middleware.RequireLoginAPI())

	// ★ここから書きはじめる
	// api.GET("/posts", listPosts)
	// api.POST("/posts", createPosts)

	_ = api // ★URLを1つでも登録したら、この行は消す
}
