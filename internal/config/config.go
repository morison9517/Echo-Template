// =============================================================================
// config.go = 設定を1か所に集める場所
//
//	.env(金庫) ──読む──> config.go ──渡す──> アプリの各所
//
//	コード内に直接パスワードやDB住所を書くと、変更時に全ファイルを探し回るうえ、
//	GitHubに秘密を上げてしまう。
//
// ★Goでは大文字で始まる名前だけが外から使える(Config は公開、env は非公開)。
//
//	「なぜか他のファイルから見えない」はほぼこれが原因。
//
// =============================================================================
package config

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// Config = アプリ全体の設定をまとめた入れ物。
//
// 1つの塊にしておくと、関数に渡すときも引数1個で済む。
type Config struct {
	Env         string // development / production
	Port        string // アプリが待ち受ける番号
	SecretKey   string // ログイン状態をブラウザに預けるときの割り印
	AuthEnabled bool   // ログイン機能を使うか

	// ▼ ★本番でいちばんハマる設定
	//
	//	true = HTTPSのときだけログイン状態を持ち歩く。本番は true が正解。
	//
	//	★HTTPのまま true にすると、ログインは成功しているのに次のページで
	//	  必ずログイン画面に戻される。エラーも出ないので原因が分からない。
	//	  練習中だけ .env に SECURE_COOKIES=false と書いて切る。
	SecureCookies bool

	// ▼ Nginxを前に置くときに必要な設定
	//
	//	Nginxを通すとEchoから見た相手はNginxになる。本当のアクセス元は
	//	ヘッダーに書いてあるので、「そのヘッダーを信じてよい相手」を指定する。
	//
	//	★空にすると誰も信用しない(開発中はこちら)。限定しないまま外に晒すと、
	//	  利用者がヘッダーを詐称してアクセス元を偽れてしまう。
	TrustedProxies []string

	// 利用者が上げたファイルの保存先。
	//
	//	★static と分ける理由: static は自分たちが用意したものでGitに入れる。
	//	  media は利用者が上げたもので、消したら戻らない。
	//	  だから本番では media だけを箱の外の保管庫に置く。
	UploadDir string

	DBHost     string
	DBPort     string
	DBName     string
	DBUser     string
	DBPassword string
}

// Load = .env を読んで Config を組み立てて返す。起動時に1回だけ呼ぶ。
//
// ★*Config の * は「中身のコピーではなく置き場所(住所)を渡す」の意味。
//
//	住所を渡せば全員が同じ1つの設定を見るので、コピーのズレが起きない。
func Load() *Config {
	// .env が無くてもエラーにしない。
	// 本番(AWS)では .env ではなくサーバー側の環境変数を直接使うため。
	if err := godotenv.Load(); err != nil {
		log.Println("[config] .env が見つかりませんでした(環境変数を直接使います)")
	}

	cfg := &Config{
		Env:         env("APP_ENV", "development"),
		Port:        env("PORT", "1323"),
		SecretKey:   env("SECRET_KEY", "dev-secret-key-change-me"),
		AuthEnabled: envBool("AUTH_ENABLED", true),

		SecureCookies:  envBool("SECURE_COOKIES", true),
		TrustedProxies: envList("TRUSTED_PROXIES"),
		UploadDir:      env("UPLOAD_DIR", "media"),

		DBHost:     env("DB_HOST", "db"),
		DBPort:     env("DB_PORT", "3306"),
		DBName:     env("DB_NAME", "hack_app"),
		DBUser:     env("DB_USER", "hack_user"),
		DBPassword: env("DB_PASSWORD", "hack_password"),
	}

	// ★本番で割り印が初期値のままなら、起動しない。
	//   初期値はGitHubで誰でも読めるので、そのまま動かすとログイン状態を偽造される。
	//   警告だけだとログに流れて見落とすため、止めて気づかせる。
	if cfg.IsProduction() && isDefaultSecretKey(cfg.SecretKey) {
		log.Fatalln("[起動失敗] 本番なのに SECRET_KEY が初期値のままです。" +
			".env の SECRET_KEY を長いランダムな文字列に変えてください(docs/DEPLOY.md)")
	}

	return cfg
}

// isDefaultSecretKey = 割り印が「配っている初期値」のままか。
//
// 初期値は2つある(.env が無いときの値と、.env.example に書いてある値)。
func isDefaultSecretKey(key string) bool {
	return key == "dev-secret-key-change-me" ||
		key == "please-change-this-to-a-long-random-string"
}

// IsProduction = 本番モードかどうか。
//
// func (c *Config) ... と書くと、cfg.IsProduction() と呼べるようになる。
func (c *Config) IsProduction() bool {
	return c.Env == "production"
}

// DSN = DBへの接続文字列を組み立てる。
//
//	ユーザー:パスワード@tcp(住所:ポート)/DB名?オプション
//
// parseTime=true … DBの日時をGoの時刻として受け取る(無いと文字列のまま届く)
// charset=utf8mb4 … 絵文字も扱える指定(utf8 だと絵文字でエラーになる)
// loc=Asia%2FTokyo … 日本時間で扱う(無いと9時間ずれる)
func (c *Config) DSN() string {
	return fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=true&loc=Asia%%2FTokyo",
		c.DBUser, c.DBPassword, c.DBHost, c.DBPort, c.DBName,
	)
}

// env = 環境変数を読む。無ければ2つ目の値を使う。
func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

// envList = カンマ区切りの環境変数を一覧にする。空なら nil。
//
// nil は「誰も信用しない」の意味になる。
//
//	TRUSTED_PROXIES=172.18.0.0/16   → ["172.18.0.0/16"]
//	TRUSTED_PROXIES=(未設定)         → nil
func envList(key string) []string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return nil
	}

	var out []string
	for _, part := range strings.Split(value, ",") {
		if p := strings.TrimSpace(part); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// envBool = .env の "true"/"false" という文字を、Goの true/false に変換する。
//
// ★.env に書けるのは文字だけなので、"false" をそのまま使うと
//
//	「中身のある文字 = true」と判定されてしまう。その事故を防ぐ。
func envBool(key string, fallback bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}
