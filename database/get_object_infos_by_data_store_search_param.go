package database

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	datastore_types "github.com/PretendoNetwork/nex-protocols-go/v2/datastore/types"
	"github.com/lib/pq"
)

const dataTypeWildcard = 0xFFFF

// GetObjectInfosByDataStoreSearchParam backs DataStore::SearchObject. It is a
// plain filtered scan of datastore.objects by data type, tags and owner.
func GetObjectInfosByDataStoreSearchParam(param datastore_types.DataStoreSearchParam, pid types.PID) ([]datastore_types.DataStoreMetaInfo, uint32, *nex.Error) {
	var pqErr *pq.Error

	dataTypes := make([]int32, 0, len(param.DataTypes))
	for _, dataType := range param.DataTypes {
		dataTypes = append(dataTypes, int32(dataType))
	}
	tags := make([]string, 0, len(param.Tags))
	for _, tag := range param.Tags {
		tags = append(tags, string(tag))
	}
	ownerIDs := make([]int64, 0, len(param.OwnerIDs))
	for _, owner := range param.OwnerIDs {
		ownerIDs = append(ownerIDs, int64(owner))
	}

	singleDataType := int32(param.DataType)
	filterBySingleDataType := singleDataType != dataTypeWildcard

	limit := int32(param.ResultRange.Length)
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	offset := int32(param.ResultRange.Offset)

	rows, err := Postgres.Query(`
		SELECT
			data_id, upload_completed, deleted, COALESCE(owner, 0),
			COALESCE(size, 0), COALESCE(name, ''), COALESCE(data_type, 0),
			meta_binary, COALESCE(permission, 0), permission_recipients,
			COALESCE(delete_permission, 0), delete_permission_recipients,
			COALESCE(flag, 0), COALESCE(period, 0), COALESCE(refer_data_id, 0),
			tags, COALESCE(persistence_slot_id, 0), extra_data,
			access_password, update_password,
			COALESCE(creation_date, to_timestamp(0)),
			COALESCE(update_date, to_timestamp(0))
		FROM datastore.objects
		WHERE upload_completed = true AND deleted = false
			AND (NOT $1::boolean OR COALESCE(data_type, 0) = $2)
			AND (cardinality($3::int[]) = 0 OR COALESCE(data_type, 0) = ANY($3::int[]))
			AND (cardinality($4::text[]) = 0 OR tags && $4::text[])
			AND (cardinality($5::bigint[]) = 0 OR COALESCE(owner, 0) = ANY($5::bigint[]))
		ORDER BY update_date DESC OFFSET $6 LIMIT $7
	`, filterBySingleDataType, singleDataType, pq.Array(dataTypes), pq.Array(tags), pq.Array(ownerIDs), offset, limit)
	if errors.Is(err, sql.ErrNoRows) || (errors.As(err, &pqErr) && pqErr.SQLState() == "42P01") {
		return nil, 0, nil
	}
	if err != nil {
		fmt.Printf("[SearchObject] query error: %s\n", err.Error())
		return nil, 0, nil
	}
	defer rows.Close()

	results, err := parseObjectList(rows)
	if err != nil {
		fmt.Printf("[SearchObject] parse error: %s\n", err.Error())
		return nil, 0, nil
	}

	return results, uint32(len(results)), nil
}
