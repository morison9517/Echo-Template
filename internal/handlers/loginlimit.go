// =============================================================================
// loginlimit.go = ログインを何度も失敗した相手を、しばらく締め出す仕掛け
//
//	パスワードは試し放題だといつか当たる(総当たり)。1回ずつは正しい手順なので、
//	パスワードの中身では防げない。「短い時間に何度も失敗している」回数で止める。
//
//	★公開するサイトのログイン画面は必ず機械に試される。
//	  公開した翌日のログに、知らないIPからの試行が並ぶ。
//
// ▼ 決めごと(下の定数で変えられる)
//
//	5回続けて失敗すると、そのアクセス元は15分間ログインできない。
//	1回でも成功すれば帳消し。最後の失敗から15分経つと記録も消える。
//
// ▼ 記録の置き場所は models/loginattempt.go(DBの表)
//
//	★アプリの中の変数ではなくDBに置いてある。理由はそちらに書いてある。
//
// ▼ 消したいとき
//
//	このファイルと models/loginattempt.go を消し、database.go の Migrate() と
//	auth.go の loginSubmit から呼び出しを外す。
//
// =============================================================================
package handlers

import (
	"log"
	"time"

	"case_echo/internal/database"
	"case_echo/internal/models"
)

// ★ここを変えれば厳しさが変わる。
const (
	// ★3回まで下げると、打ち間違いで普通の利用者が締め出される。
	maxLoginAttempts = 5

	// ★長くしすぎないこと。総当たりは「1分間に何回試せるか」で成否が決まるので
	//   15分でも十分に割に合わなくなる。長いほど本人が戻れない時間が伸びるだけ。
	loginLockDuration = 15 * time.Minute

	// ★これが無いと、何日も前の1回の失敗が残り続け、
	//   久しぶりに来て4回間違えただけで締め出される。
	loginAttemptWindow = 15 * time.Minute
)

// loginLockRemaining = あと何分締め出されているか。0 なら締め出されていない。
func loginLockRemaining(ip string) time.Duration {
	var attempt models.LoginAttempt

	if err := database.DB.Where("ip = ?", ip).First(&attempt).Error; err != nil {
		// 記録が無い = 一度も失敗していない。
		// ★DBの異常もここに来るが、その場合も通す。DBが不調なときに
		//   全員がログインできなくなるほうが困るため。
		return 0
	}

	// ★まだ締め出していない(失敗を数えている途中)。
	//   ここを飛ばすと、数えている途中の記録まで下で消してしまい、
	//   回数が5に届かず締め出しが一度も働かなくなる。
	if attempt.LockedUntil == nil {
		return 0
	}

	remaining := time.Until(*attempt.LockedUntil)
	if remaining <= 0 {
		database.DB.Delete(&attempt)
		return 0
	}

	return remaining
}

// recordLoginFailure = 失敗を1回数える。5回目で締め出す。
func recordLoginFailure(ip string) {
	sweepOldLoginAttempts()

	var attempt models.LoginAttempt

	if err := database.DB.Where("ip = ?", ip).First(&attempt).Error; err != nil {
		attempt = models.LoginAttempt{IP: ip}
	}

	attempt.Count++
	attempt.LastFailureAt = time.Now()

	if attempt.Count >= maxLoginAttempts {
		// ★Truncate で秒未満を切り捨てる。
		//   DBの時刻の列は秒までしか持てず、秒未満は四捨五入される。
		//   切り上がると残り時間が15分を超え、画面に「あと約16分」と出る。
		until := time.Now().Add(loginLockDuration).Truncate(time.Second)
		attempt.LockedUntil = &until

		// ★締め出したことはログに残す。これが無いと「ログインできない」と
		//   言われたときに、締め出したのか設定が壊れたのか切り分けられない。
		log.Printf("[login] %s を %v 締め出しました(%d回失敗)",
			ip, loginLockDuration, attempt.Count)
	}

	// Save = 番号があれば更新、無ければ追加。
	if err := database.DB.Save(&attempt).Error; err != nil {
		// ★数えられなくてもログインは続けさせる(DBの不調を障害にしない)。
		log.Printf("[login] 失敗回数を記録できませんでした: %v", err)
	}
}

// clearLoginFailures = 失敗の記録を消す。ログイン成功時に呼ぶ。
func clearLoginFailures(ip string) {
	database.DB.Where("ip = ?", ip).Delete(&models.LoginAttempt{})
}

// remainingLoginAttempts = あと何回間違えたら締め出されるか。画面表示用。
//
// ★回数を教えてよい。機械は表示されなくても総当たりを続けるので隠しても
//
//	効果が無く、普通の利用者には「次で締め出される」と分かるほうが親切。
func remainingLoginAttempts(ip string) int {
	var attempt models.LoginAttempt

	if err := database.DB.Where("ip = ?", ip).First(&attempt).Error; err != nil {
		return maxLoginAttempts
	}

	remaining := maxLoginAttempts - attempt.Count
	if remaining < 0 {
		return 0
	}
	return remaining
}

// sweepOldLoginAttempts = 古くなった記録を捨てる。
//
// ★捨てないと、IPを変えながら試された分だけ表が際限なく大きくなる。
//
//	締め出し中のものは明けるまで残す。
func sweepOldLoginAttempts() {
	cutoff := time.Now().Add(-loginAttemptWindow)

	database.DB.
		Where("last_failure_at < ?", cutoff).
		Where("locked_until IS NULL OR locked_until < ?", time.Now()).
		Delete(&models.LoginAttempt{})
}
