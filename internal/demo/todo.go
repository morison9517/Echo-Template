// =============================================================================
// デモ用のデータの形。
//
//	★この表(demo_todos)は、開発モードで demo.Register() が呼ばれたときだけ
//	  作られます。本番のDBには作られません。
//
//	自分たちの表は internal/models/ に書きます。書き方の見本としてどうぞ。
//	(新しいモデルを作ったら internal/database/database.go の Migrate() の
//	 リストに1行足すこと。忘れると表が作られない)
//
//	★項目名(json:"...")を変えるときは internal/demo/page.html のJSも一緒に直す。
//
// ▼ ★デモは本番のモデルに一切ぶら下がりません
//
//	以前ここには User への紐付け(外部キー)がありましたが、外しました。
//	紐付けがあると、デモ用の表から本番の users 表に向けて
//	「fk_todos_user」という制約がDBに作られてしまいます。
//	そうなると、自分たちがユーザー機能を作り替えるときに、
//	消したはずのデモがDBの中から邪魔をしてきます。
//
//	デモはデモだけで完結させ、本番側には何も残さない方針にしています。
//	紐付けの書き方の見本は internal/models/user.go の末尾にあります。
//
// =============================================================================
package demo

import (
	"time"
)

// Todo = やることリスト1件分(デモ)。
type Todo struct {
	ID     uint   `gorm:"primaryKey" json:"id"`
	Title  string `gorm:"size:200;not null" json:"title"`
	IsDone bool   `gorm:"not null;default:false" json:"is_done"`

	CreatedAt time.Time `json:"created_at"`
}

// TableName = この構造体が使うDBの表の名前。
//
// ★demo_ を付けている理由
//
//	"todos" のままだと、開発する人が自分たちのTodoを作ったときに
//	同じ表を取り合ってしまう。デモが名前を1つ占領しないよう、必ず demo_ を付ける。
func (Todo) TableName() string {
	return "demo_todos"
}
