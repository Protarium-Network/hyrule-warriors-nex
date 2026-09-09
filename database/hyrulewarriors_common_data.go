package database

import (
	"database/sql"
	"time"

	"github.com/PretendoNetwork/nex-go/v2/types"
)

func HyruleWarriorsGetCommonData(pid types.PID, uniqueID types.UInt64) (types.Buffer, error) {
	var data []byte
	var err error

	if uniqueID == 0 {
		err = Postgres.QueryRow(`
			SELECT common_data
			FROM hyrulewarriors_common_datas
			WHERE owner_pid = $1
			ORDER BY updated_at DESC
			LIMIT 1
		`, int64(pid)).Scan(&data)
	} else {
		err = Postgres.QueryRow(`
			SELECT common_data
			FROM hyrulewarriors_common_datas
			WHERE owner_pid = $1 AND unique_id = $2
		`, int64(pid), int64(uniqueID)).Scan(&data)
	}

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, err
		}
		return nil, err
	}

	return types.NewBuffer(data), nil
}

func HyruleWarriorsUploadCommonData(pid types.PID, uniqueID types.UInt64, commonData types.Buffer) error {
	now := time.Now().Unix()
	_, err := Postgres.Exec(`
		INSERT INTO hyrulewarriors_common_datas (common_data, unique_id, owner_pid, updated_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (unique_id, owner_pid) DO UPDATE SET
			common_data = EXCLUDED.common_data,
			updated_at = EXCLUDED.updated_at
	`,
		[]byte(commonData),
		int64(uniqueID),
		int64(pid),
		now,
	)
	return err
}
