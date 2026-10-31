package domain

import "time"

// User represents a registered reader on the platform
type User struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Password  string    `json:"-"`
	Realm     string    `json:"realm"` // Phàm Nhân, Luyện Khí, Trúc Cơ, Kim Đan, Nguyên Anh, Hóa Thần...
	Exp       int       `json:"exp"`   // Tu vi (tăng khi đọc chương)
	AvatarURL string    `json:"avatar_url"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// RealmLevels defines the cultivation hierarchy for readers
func CalculateRealm(exp int) (string, string) {
	if exp < 100 {
		return "Phàm Nhân", "Khởi đầu con đường tầm đạo"
	} else if exp < 500 {
		return "Luyện Khí Kỳ", "Dẫn khí nhập thể, gân cốt dẻo dai"
	} else if exp < 1500 {
		return "Trúc Cơ Kỳ", "Đúc thành đạo cơ, trường thọ trăm năm"
	} else if exp < 3500 {
		return "Kim Đan Kỳ", "Một hạt kim đan nuốt vào bụng, mệnh ta do ta chẳng do trời"
	} else if exp < 7500 {
		return "Nguyên Anh Kỳ", "Nguyên anh xuất khiếu, ngao du thiên địa"
	} else if exp < 15000 {
		return "Hóa Thần Kỳ", "Hóa thần nhập đạo, lĩnh ngộ thiên địa quy tắc"
	} else if exp < 30000 {
		return "Luyện Hư Kỳ", "Phản phác quy chân, chưởng khống hư không"
	} else if exp < 60000 {
		return "Hợp Thể Kỳ", "Thân dung thiên địa, vạn pháp bất xâm"
	} else if exp < 100000 {
		return "Đại Thừa Kỳ", "Viên mãn đạo quả, ngóng trông tiên môn"
	}
	return "Độ Kiếp Kỳ", "Vượt chín tầng thiên kiếp, đắc đạo phi thăng"
}

