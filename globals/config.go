package globals

// NEX configuration for "Hyrule Warriors" (Wii U, Koei Tecmo / Nintendo,
// 2014; internal codename "Project Z").
//
// GameServerID and AccessKey come from kinnay.github.io's public Wii U NEX
// game database (entry "Hyrule Warriors", id 269995264). The id as 8 hex
// digits is the value the console passes to
// nn::act::AcquireNexServiceToken and the game logs as
// "GameID:%08x" - confirmed present in the retail RPX (ProjectZ_r.rpx).
const (
	GameServerID = "1017cd00" // 269995264
	AccessKey    = "7fcc1f7c"

	// PRUDP library version reported by both endpoints. Hyrule Warriors
	// ships NEX 3.8.13 (kinnay build tag "build:3_8_13_2004_0",
	// branch "origin/release/ngs/3.8.x.200x").
	//
	// Unlike the 3.4.x Mario & Sonic servers this was templated from, no
	// library version is overridden at runtime: at 3.8.13 the Ranking
	// library already serializes RankingRankData.UpdateTime (the reason
	// those servers bumped Ranking to 3.6.0) and MatchMaking already uses
	// the modern MatchmakeSession / search-criteria layout (the reason
	// they pinned MatchMaking to 3.3.0).
	NEXMajor = 3
	NEXMinor = 8
	NEXPatch = 13
)
