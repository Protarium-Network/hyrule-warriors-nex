package database

// Ranking mode values shared by the ranking queries. These match
// NEX's own RankingMode enum (Range = 0, User = 4); Near (1) and
// friend-range (2) are handled per-title in hyrulewarriors_get_rankings.go.
const (
	rankingModeRange = 0
	rankingModeUser  = 4
	maxRankingRows   = 1000
)
