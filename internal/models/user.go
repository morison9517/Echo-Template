// =============================================================================
// models = データの形(DBの表)を決める場所
//
//	1つの struct = DBの1つの表。1行 = 1件のデータ。
//	    type User struct  →  users 表
//
//	この設計図があるおかげで、SQLを書かずに
//	    database.DB.Find(&users)        → 全件取得
//	    database.DB.Create(&user)       → 1件追加
//	と書ける。
//
// ▼ `gorm:"..."` `json:"..."` の部分(タグ)について
//
//	項目名の右に付いている付箋のようなもの。
//	    gorm: DBの表を作るときの指示(文字数、重複禁止、など)
//	    json: JavaScriptに渡すときの名前
//	Goの項目名は大文字始まり(Username)だが、JSONでは小文字(username)に
//	したいので、json タグで名前を指定している。
//
// ▼ ★大文字始まりでないと動かない
//
//	項目名が小文字始まりだと、GORMもJSONも画面のHTMLもその項目を読めない。
//	「保存されない」「画面に出ない」の原因のほとんどがこれ。
//
// =============================================================================
package models

import (
	"time"

	"golang.org/x/crypto/bcrypt"
)

// User = ユーザー1人分。
type User struct {
	// primaryKey = この番号で1行を必ず特定できる印。自動で1,2,3…と振られる。
	ID uint `gorm:"primaryKey" json:"id"`

	// size:80        … MySQLでは文字の最大長を決める必要がある
	// uniqueIndex    … 同じ名前を2人が使えないようにする + 検索を速くする目次
	// not null       … 空を許さない
	Username string `gorm:"size:80;uniqueIndex;not null" json:"username"`

	// *string の * は「値が無い(未入力)状態を持てる」という意味。
	// メールアドレスは任意入力なので、空とも違う「未入力」を表せるようにしている。
	Email *string `gorm:"size:255;uniqueIndex" json:"email,omitempty"`

	// ★パスワードそのものは保存しない。元に戻せない形(ハッシュ)にして保存する。
	//   DBが盗まれてもパスワードは復元できない。変換後は長くなるので255文字。
	//
	// json:"-" = 「JSONには絶対に含めない」という指定。
	//   これを忘れると、APIの返事にパスワードが混ざって画面まで流れ出る。
	PasswordHash string `gorm:"size:255;not null" json:"-"`

	// CreatedAt という名前にしておくと、GORMが保存時に自動で現在時刻を入れる。
	// アプリ側で入れ忘れる事故がなくなる(お決まりの名前)。
	CreatedAt time.Time `json:"created_at"`
}

// TableName = この設計図が対応する表の名前。
//
// 書かなくてもGORMが "users" と決めるが、探しにくいので明示しておく。
func (User) TableName() string {
	return "users"
}

// SetPassword = 生のパスワードを、元に戻せない形に変換して保存する。
//
// ★(u *User) のアスタリスクは「本人を書き換える」の意味。
//
//	付け忘れるとコピーを触るだけになり、パスワードが空のままになる。
func (u *User) SetPassword(password string) error {
	// bcrypt = パスワード専用の変換方式。
	// わざと計算に時間がかかるように作られていて、総当たり攻撃をやりにくくしている。
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.PasswordHash = string(hash)
	return nil
}

// CheckPassword = 入力されたパスワードが正しいか確かめる。
//
// 保存してあるのは戻せない形なので、「入力を同じ方式で変換して見比べる」ことで確認する。
func (u *User) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password))
	return err == nil
}

// ★ここから自分たちの表を書きはじめる
//
//	作ったら database.go の Migrate() に1行足すこと。忘れると表が作られない。
//
//	type Post struct {
//	    ID    uint   `gorm:"primaryKey" json:"id"`
//	    Title string `gorm:"size:200;not null" json:"title"`
//
//	    // *uint にすると「持ち主なし」も許される(ログイン機能をOFFにしても動く)。
//	    UserID *uint `gorm:"index" json:"user_id,omitempty"`
//
//	    // Preload("User") で post.User に作者が入る。
//	    // OnDelete:CASCADE = ユーザーが消えたら投稿も一緒に消す。
//	    User *User `gorm:"constraint:OnDelete:CASCADE" json:"-"`
//
//	    CreatedAt time.Time `json:"created_at"`
//	}
