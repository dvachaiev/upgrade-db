package db

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
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
	len_a, len_v := len(a.seq), len(v.seq)

	var max_len int
	if len_a > len_v {
		max_len = len_a
	} else {
		max_len = len_v
	}

	for i := 0; i < max_len; i++ {
		el_a, el_v := 0, 0
		if i < len_a {
			el_a = a.seq[i]
		}
		if i < len_v {
			el_v = v.seq[i]
		}

		switch {
		case el_v < el_a:
			return true
		case el_v > el_a:
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

func CreateVersionTable(db *sql.DB) error {
	const query = "CREATE TABLE IF NOT EXISTS `_db_version` ( " +
		"`version` VACHAR ( 16 ) NOT NULL UNIQUE," +
		"`applied_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP )"
	_, err := db.Exec(query)
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
	const query = "INSERT INTO `_db_version` (`version`) VALUES ($1)"

	_, err := db.Exec(query, version.String())
	return err
}

func UpgradeVersion(db *sql.DB, version Version, files []string) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback() // nolint

	for _, fname := range files {
		fmt.Println("\t\tApplying", fname)

		buf, err := ioutil.ReadFile(fname)
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
