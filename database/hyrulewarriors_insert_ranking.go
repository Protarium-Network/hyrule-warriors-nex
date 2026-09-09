package database

import (
	"fmt"
	"time"

	"github.com/PretendoNetwork/nex-go/v2/types"
	ranking_types "github.com/PretendoNetwork/nex-protocols-go/v2/ranking/types"
	"github.com/Protarium-Network/hyrule-warriors-nex/globals"
)

func HyruleWarriorsInsertRankingByPIDAndRankingScoreData(pid types.PID, rankingScoreData ranking_types.RankingScoreData, uniqueID types.UInt64) error {
	if uint8(rankingScoreData.OrderBy) > 1 {
		return fmt.Errorf("invalid ranking order %d", uint8(rankingScoreData.OrderBy))
	}
	if uint8(rankingScoreData.UpdateMode) > 1 {
		return fmt.Errorf("invalid ranking update mode %d", uint8(rankingScoreData.UpdateMode))
	}

	now := time.Now().Unix()
	tx, err := Postgres.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
		INSERT INTO hyrulewarriors_ranking_categories (category, order_by, created_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (category) DO NOTHING
	`,
		int64(rankingScoreData.Category),
		int16(rankingScoreData.OrderBy),
		now,
	)
	if err != nil {
		return err
	}

	result, err := tx.Exec(`
		INSERT INTO hyrulewarriors_rankings (
			owner_pid, unique_id, category, score, order_by,
			update_mode, groups, param, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (unique_id, owner_pid, category) DO UPDATE SET
			score = EXCLUDED.score,
			order_by = EXCLUDED.order_by,
			update_mode = EXCLUDED.update_mode,
			groups = EXCLUDED.groups,
			param = EXCLUDED.param,
			updated_at = EXCLUDED.updated_at
		WHERE
			EXCLUDED.update_mode = 1
			OR (hyrulewarriors_rankings.order_by = 0 AND EXCLUDED.score < hyrulewarriors_rankings.score)
			OR (hyrulewarriors_rankings.order_by = 1 AND EXCLUDED.score > hyrulewarriors_rankings.score)
	`,
		int64(pid),
		int64(uniqueID),
		int64(rankingScoreData.Category),
		int64(rankingScoreData.Score),
		int16(rankingScoreData.OrderBy),
		int16(rankingScoreData.UpdateMode),
		[]byte(rankingScoreData.Groups),
		int64(rankingScoreData.Param),
		now,
	)
	if err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	globals.Logger.Infof("[HYRULEWARRIORS RANKING] upload PID=%d category=%d score=%d stored=%t",
		uint64(pid), uint32(rankingScoreData.Category), uint32(rankingScoreData.Score), rowsAffected > 0)

	return nil
}
