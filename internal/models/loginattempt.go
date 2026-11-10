// =============================================================================
// loginattempt.go = ログインの失敗を数えておく表
//
//	数え方と締め出しの判断は internal/handlers/loginlimit.go。
//	ここは置き場所の形だけを決めている。
//
// ▼ ★なぜアプリの中の変数ではなくDBに置くのか
//
//	本番ではアプリが複数のプロセスで動く(Echo版は1つだが、Flask版とDjango版は
//	gunicorn が3つ立てる)。変数に数えるとプロセスごとに別々に数えるので、
//	「5回まで」のつもりが実質15回になる。
//	★見た目は動いているのに効いていない、という最悪の形になる。
//	DBなら何プロセスでも、サーバーを増やしても正しく数えられる。
//
// =============================================================================
package models

import "time"

// LoginAttempt = あるアクセス元(IP)の失敗の記録1件。
type LoginAttempt struct {
	ID uint `gorm:"primaryKey" json:"id"`

	// ★「アカウントごと」ではなく「アクセス元ごと」に数える。
	//   アカウントごとだと、他人がわざと5回間違えるだけで本人が締め出される。
	//
	// size:45 … IPv6 のいちばん長い形が45文字。足りないと保存に失敗する。
	IP string `gorm:"size:45;uniqueIndex;not null" json:"ip"`

	// 続けて失敗した回数。1回でも成功したら記録ごと消える。
	Count int `gorm:"not null;default:0" json:"count"`

	// 古い記録を捨てる判断に使う。
	LastFailureAt time.Time `json:"last_failure_at"`

	// ★*time.Time の * は「まだ無い状態を持てる」の意味。
	//   失敗を数えている途中(まだ締め出していない)を表すために必要。
	//   普通の time.Time にすると未設定が「西暦1年」になり、
	//   必ず「もう過ぎている」と判定されて締め出しが働かない。
	LockedUntil *time.Time `json:"locked_until,omitempty"`
}

func (LoginAttempt) TableName() string {
	return "login_attempts"
}
