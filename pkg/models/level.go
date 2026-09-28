package models

// Level - statik jadval, admin panel orqali CRUD qilinadi (foydalanuvchi bilan
// kelishilgan qaror: dinamik formula emas, aniq XP chegaralari).
// Slug - level'ning o'zgarmas kodi (masalan "beginner", "resident"), mobil/frontend
// tomonda ikonka/rang/tarjima shu bo'yicha bog'lanadi (level_number/title o'zgarsa ham slug qolaveradi).

// XPReward/CoinReward - shu levelga yetgan foydalanuvchi case yakunlaganda oladigan
// bazaviy mukofot (case.difficulty koeffitsienti va health_percent bilan yana
// ko'paytiriladi - ko'ring [[SimulationService.Finish]]). Har bir level o'z
// mukofot miqdorini belgilaydi - masalan Level 5 (Mutaxassis) Level 1'dan ko'proq.

type LevelCreateReq struct {
	LevelNumber   int    `json:"level_number"`
	Slug          string `json:"slug"`
	RequiredXP    int    `json:"required_xp"`
	Title         string `json:"title"`
	BadgeImageUrl string `json:"badge_image_url"`
	XPReward      int    `json:"xp_reward"`
	CoinReward    int    `json:"coin_reward"`
}

type LevelUpdateReq struct {
	Id            string `json:"id"`
	LevelNumber   int    `json:"level_number"`
	Slug          string `json:"slug"`
	RequiredXP    int    `json:"required_xp"`
	Title         string `json:"title"`
	BadgeImageUrl string `json:"badge_image_url"`
	XPReward      int    `json:"xp_reward"`
	CoinReward    int    `json:"coin_reward"`
}

type LevelRes struct {
	Id            *string `json:"id"`
	LevelNumber   *int    `json:"level_number"`
	Slug          *string `json:"slug"`
	RequiredXP    *int    `json:"required_xp"`
	Title         *string `json:"title"`
	BadgeImageUrl *string `json:"badge_image_url"`
	XPReward      *int    `json:"xp_reward"`
	CoinReward    *int    `json:"coin_reward"`
}

// AddPointsReq - keys yakunlanganda XP/Coin qo'shish (UserI.AddPoints).
// Service qatlamida: coins.Delta faqat muvaffaqiyatli yakunlanganda musbat bo'ladi
// (BACKEND_STRUCTURE.md 6-bo'lim: "muvaffaqiyatsiz yechilsa tanga kamaytirilmaydi").
type AddPointsReq struct {
	UserId      string `json:"user_id"`
	XPDelta     int    `json:"xp_delta"`
	CoinsDelta  int    `json:"coins_delta"`
	Reason      string `json:"reason"`
	ReferenceId string `json:"reference_id"`
}
