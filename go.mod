// =============================================================================
// go.mod = 買い物リスト(このアプリが使っているライブラリの一覧)
//
//	go.sum … レシート。全員が「まったく同じ中身」を使うための記録。
//	★どちらも必ずGitHubに上げる。上げないと他の人の環境でビルドできない。
//
//	ライブラリを追加するときは手で書かない。次のコマンドを打てば自動で追記される:
//	    docker compose exec web go get <ライブラリ名>
//
//	下の「// indirect」が付いているものは、上のライブラリが内部で使っている部品。
//	自動で管理されるので、気にしなくてよい。
// =============================================================================

module case_echo

// このアプリを動かすのに必要なGoのバージョン。
go 1.26.0

require (
	github.com/gorilla/sessions v1.4.0
	github.com/joho/godotenv v1.5.1
	github.com/labstack/echo/v5 v5.4.4
	golang.org/x/crypto v0.58.0
	gorm.io/driver/mysql v1.6.0
	gorm.io/gorm v1.31.2
)

require (
	filippo.io/edwards25519 v1.2.0 // indirect
	github.com/go-sql-driver/mysql v1.10.1 // indirect
	github.com/gorilla/securecookie v1.1.2 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	golang.org/x/text v0.44.0 // indirect
	golang.org/x/time v0.16.0 // indirect
)
