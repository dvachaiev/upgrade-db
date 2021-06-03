package db

import (
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/afero"
)

type Version struct {
	seq []int
}

func (v Version) String() string {
	strSeq := make([]string, 0, 3)

	for _, el := range v.seq {
		strSeq = append(strSeq, strconv.Itoa(el))
	}

	return strings.Join(strSeq, ".")
}

func (v Version) Less(a Version) bool {
	lenA, lenV := len(a.seq), len(v.seq)

	var maxLen int
	if lenA > lenV {
		maxLen = lenA
	} else {
		maxLen = lenV
	}

	for i := 0; i < maxLen; i++ {
		elA, elV := 0, 0

		if i < lenA {
			elA = a.seq[i]
		}

		if i < lenV {
			elV = v.seq[i]
		}

		switch {
		case elV < elA:
			return true
		case elV > elA:
			return false
		}
	}

	return true
}

func (v Version) IsZero() bool {
	return len(v.seq) == 0
}

func (v Version) Equal(a Version) bool {
	lenA, lenV := len(a.seq), len(v.seq)

	var maxLen int
	if lenA > lenV {
		maxLen = lenA
	} else {
		maxLen = lenV
	}

	for i := 0; i < maxLen; i++ {
		elA, elV := 0, 0

		if i < lenA {
			elA = a.seq[i]
		}

		if i < lenV {
			elV = v.seq[i]
		}

		if elV != elA {
			return false
		}
	}

	return true
}

func ParseVersion(version string) (Version, error) {
	tmp := make([]int, 0, 3)

	for _, el := range strings.Split(version, ".") {
		val, err := strconv.Atoi(el)
		if err != nil {
			return Version{}, err
		}

		tmp = append(tmp, val)
	}

	return Version{tmp}, nil
}

func PrepareVersionTable(db *sql.DB) error {
	if err := createVersionTable(db); err != nil {
		return err
	}

	if err := removeUniqIndex(db); err != nil {
		return err
	}

	return nil
}

func createVersionTable(db *sql.DB) error {
	const query = "CREATE TABLE IF NOT EXISTS `_db_version` ( " +
		"`version` VACHAR ( 16 ) NOT NULL," +
		"`applied_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP )"

	_, err := db.Exec(query)

	return err
}

func removeUniqIndex(db *sql.DB) error {
	const (
		queryCheck = `SELECT ii.name AS column_name FROM sqlite_master AS m, pragma_index_list(m.name) AS il, pragma_index_info(il.name) AS ii
	  WHERE m.tbl_name = '_db_version' AND ii.name = 'version' AND m.type = 'table' AND il.origin = 'u'`

		queryUpdate = `
		BEGIN TRANSACTION;

		ALTER TABLE _db_version RENAME TO _old_db_version;

		CREATE TABLE IF NOT EXISTS _db_version
		( version VACHAR ( 16 ) NOT NULL,
		  applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);

		INSERT INTO _db_version SELECT * FROM _old_db_version;

		DROP TABLE _old_db_version;

		COMMIT;`
	)

	var res interface{}

	if err := db.QueryRow(queryCheck).Scan(&res); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}

		return err
	}

	_, err := db.Exec(queryUpdate)

	return err
}

func GetVersion(db *sql.DB) (Version, error) {
	const query = "SELECT `version` FROM `_db_version` ORDER BY `rowid` DESC LIMIT 1"

	var strVer string

	row := db.QueryRow(query)

	switch err := row.Scan(&strVer); err {
	default:
		return Version{}, err
	case sql.ErrNoRows:
		return Version{[]int{0, 0, 0}}, nil
	case nil:
		return ParseVersion(strVer)
	}
}

func setVersion(db *sql.Tx, version Version) error {
	const query = "INSERT OR REPLACE INTO `_db_version` (`version`) VALUES ($1)"

	_, err := db.Exec(query, version.String())

	return err
}

func ApplyVersion(db *sql.DB, version Version, fs afero.Fs, files []string) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	defer tx.Rollback() // nolint

	for _, fname := range files {
		fmt.Println("\t\tApplying", fname)

		buf, err := afero.ReadFile(fs, fname)
		if err != nil {
			return err
		}

		_, err = tx.Exec(string(buf))
		if err != nil {
			return err
		}
	}

	err = setVersion(tx, version)
	if err == nil {
		err = tx.Commit()
	}

	return err
}
