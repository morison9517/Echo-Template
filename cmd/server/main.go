// =============================================================================
// main.go = アプリを起動する入口
//
// ▼ ★Goの決まり
//
//	package main と func main() があるファイルが「実行できるプログラム」になる。
//	docker compose up をすると、最終的にここが動く。
//
// ▼ ここは基本的に触らない
//
//	機能を足すときに触るのは handlers/ と models/ と web/templates/。
//
// ▼ やっていること(4ステップだけ)
//
//  1. .env を読む
//  2. DBに繋いで、表を用意する
//  3. URLの受付を組み立てる
//  4. 待ち受け開始
//
// =============================================================================
package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/labstack/echo/v5"

	"case_echo/internal/config"
	"case_echo/internal/database"
	"case_echo/internal/router"
)

func main() {
	// ▼ -reset-db を付けて起動すると、DBを作り直して終了する。
	//
	//	docker compose exec web go run ./cmd/server -reset-db
	//
	//	★中のデータは全部消える。列の型を変えて形が合わなくなったときに使う。
	resetDB := flag.Bool("reset-db", false, "DBの表を作り直す(中のデータは全部消える)")
	flag.Parse()

	// --- 1. 設定を読む ---
	cfg := config.Load()

	// --- 2. DBに繋ぐ ---
	if err := database.Connect(cfg); err != nil {
		// log.Fatalf = メッセージを出してプログラムを終了する。
		// DBに繋がらないまま動かしても意味がないので、ここで止める。
		log.Fatalf("[起動失敗] DBに接続できません: %v", err)
	}

	if *resetDB {
		if err := database.Reset(); err != nil {
			log.Fatalf("[reset-db] 失敗しました: %v", err)
		}
		log.Println("[reset-db] DBの表を作り直しました")
		return
	}

	// models/ の設計図どおりに表を用意する。
	// ★起動のたびに自動で実行されるので、「表を作るコマンド」を打つ必要はない。
	if err := database.Migrate(); err != nil {
		log.Fatalf("[起動失敗] 表を作れません: %v", err)
	}

	// --- 3. 受付を組み立てる ---
	e, err := router.New(cfg)
	if err != nil {
		log.Fatalf("[起動失敗] %v", err)
	}

	// --- 4. 待ち受け開始 ---
	// ★":1323" の前にホスト名を書かない(= どこからの接続も受ける)。
	//   "localhost:1323" と書くと箱の中からしか繋がらず、
	//   「起動しているのに画面が出ない」状態になる。
	addr := ":" + cfg.Port
	log.Printf("[起動] http://localhost%s で待ち受けます", addr)

	// ★止める合図(Ctrl+C や docker compose down)が来たら、処理中の依頼を
	//   最後まで返してから終わる。途中で切ると、保存の最中だった人の画面が固まる。
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// HideBanner / HidePort = Echoの起動時の飾りを出さない(上の1行で足りるため)。
	server := echo.StartConfig{Address: addr, HideBanner: true, HidePort: true}

	if err := server.Start(ctx, e); err != nil {
		log.Fatalf("[起動失敗] %v", err)
	}
}
