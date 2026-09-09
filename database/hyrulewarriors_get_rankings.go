package database

import (
	"fmt"

	"github.com/PretendoNetwork/nex-go/v2/types"
	ranking_constants "github.com/PretendoNetwork/nex-protocols-go/v2/ranking/constants"
	ranking_types "github.com/PretendoNetwork/nex-protocols-go/v2/ranking/types"
)

// Hyrule Warriors gets its own ranking tables (hyrulewarriors_rankings etc.),
// isolated from every other title's tables: nothing in NEX's ranking protocol
// scopes categories by title, so category numbers could otherwise collide
// across games sharing one database.
func HyruleWarriorsGetRankingsAndCountByCategoryAndRankingOrderParam(category types.UInt32, rankingOrderParam ranking_types.RankingOrderParam) (types.List[ranking_types.RankingRankData], uint32, error) {
	return HyruleWarriorsGetRankings(
		types.NewUInt8(rankingModeRange),
		types.NewPID(0),
		types.NewUInt64(0),
		category,
		rankingOrderParam,
	)
}

// Same ranking modes as Rio 2016: range(0), near(1) and friend-range(2). No
// friends system yet, so friend-range degrades to the user-mode query (the
// protocol's own documented fallback).
const rankingModeNearHyruleWarriors = 1
const rankingModeFriendRangeHyruleWarriors = 2

func HyruleWarriorsGetRankings(
	rankingMode types.UInt8,
	principalID types.PID,
	uniqueID types.UInt64,
	category types.UInt32,
	rankingOrderParam ranking_types.RankingOrderParam,
) (types.List[ranking_types.RankingRankData], uint32, error) {
	mode := uint8(rankingMode)
	if mode == rankingModeFriendRangeHyruleWarriors {
		mode = rankingModeUser
	}
	if mode != rankingModeRange && mode != rankingModeUser && mode != rankingModeNearHyruleWarriors {
		return nil, 0, fmt.Errorf("ranking mode %d is not supported", mode)
	}

	groupIndex := uint8(rankingOrderParam.GroupIndex)
	if groupIndex != uint8(ranking_constants.FilterGroupIndexNone) && groupIndex > 3 {
		return nil, 0, fmt.Errorf("invalid ranking group index %d", groupIndex)
	}

	orderCalculation := uint8(rankingOrderParam.OrderCalculation)
	if orderCalculation > 1 {
		return nil, 0, fmt.Errorf("invalid ranking order calculation %d", orderCalculation)
	}

	limit := uint32(rankingOrderParam.Length)
	if limit == 0 {
		limit = maxRankingRows
	}
	if limit > maxRankingRows {
		limit = maxRankingRows
	}

	offset := uint32(rankingOrderParam.Offset)
	if mode == rankingModeUser || mode == rankingModeNearHyruleWarriors {
		offset = 0
	}

	rows, err := Postgres.Query(`
		WITH base AS (
			SELECT
				r.owner_pid,
				r.unique_id,
				r.category,
				r.score,
				r.groups,
				r.param,
				r.updated_at,
				COALESCE(cd.common_data, ''::bytea) AS common_data,
				COALESCE(rc.order_by, r.order_by, 1) AS category_order
			FROM hyrulewarriors_rankings AS r
			LEFT JOIN hyrulewarriors_ranking_categories AS rc
				ON rc.category = r.category
			LEFT JOIN hyrulewarriors_common_datas AS cd
				ON cd.owner_pid = r.owner_pid AND cd.unique_id = r.unique_id
			WHERE r.category = $1
				AND CASE
					WHEN $2::integer = 255 THEN TRUE
					ELSE octet_length(r.groups) > $2::integer
						AND get_byte(r.groups, $2::integer) = $3::integer
				END
		), ranked AS (
			SELECT
				base.*,
				CASE
					WHEN $4::integer = 0 THEN RANK() OVER (
						ORDER BY
							CASE WHEN category_order = 1 THEN score END DESC,
							CASE WHEN category_order = 0 THEN score END ASC
					)
					ELSE ROW_NUMBER() OVER (
						ORDER BY
							CASE WHEN category_order = 1 THEN score END DESC,
							CASE WHEN category_order = 0 THEN score END ASC,
							updated_at ASC,
							owner_pid ASC,
							unique_id ASC
					)
				END AS ranking_order
			FROM base
		), selected AS (
			SELECT ranked.*, COUNT(*) OVER () AS total_count
			FROM ranked
			WHERE $5::integer = 0
				OR (
					$5::integer = 4
					AND owner_pid = $6
					AND ($7::bigint = 0 OR unique_id = $7)
				)
				OR (
					$5::integer = 1
					AND ranking_order BETWEEN
						(SELECT MIN(ranking_order) FROM ranked WHERE owner_pid = $6 AND ($7::bigint = 0 OR unique_id = $7)) - ($9::integer / 2)
						AND
						(SELECT MIN(ranking_order) FROM ranked WHERE owner_pid = $6 AND ($7::bigint = 0 OR unique_id = $7)) + ($9::integer / 2)
				)
		)
		SELECT
			owner_pid,
			unique_id,
			ranking_order,
			category,
			score,
			groups,
			param,
			common_data,
			updated_at,
			total_count
		FROM selected
		ORDER BY ranking_order, owner_pid, unique_id
		OFFSET $8
		LIMIT $9
	`,
		int64(category),
		int(groupIndex),
		int(uint8(rankingOrderParam.GroupNum)),
		int(orderCalculation),
		int(mode),
		int64(principalID),
		int64(uniqueID),
		int64(offset),
		int64(limit),
	)
	if err != nil {
		return nil, 0, fmt.Errorf("query category %d: %w", uint32(category), err)
	}
	defer rows.Close()

	results, totalCount, err := parseRankingDataList(rows)
	if err != nil {
		return nil, 0, fmt.Errorf("parse category %d: %w", uint32(category), err)
	}

	return results, totalCount, nil
}
