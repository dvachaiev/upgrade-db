package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/dvachaiev/upgrade-db/db"
	_ "github.com/mattn/go-sqlite3"
)

const (
	ECodeBadArgs = iota + 1
	ECodeBadDB
	ECodeBadPath
	ECodeMigrFailed
)

func listVersions(path string) ([]db.Version, error) {
	ptrn := regexp.MustCompile(`^[0-9]+(\.[0-9]+)*$`)

	var versions []db.Version

	f, err := os.Open(path)
	if err != nil {
		return versions, err
	}
	defer f.Close()

	files, err := f.Readdir(-1)
	if err != nil {
		return versions, err
	}

	for _, fObj := range files {
		if !fObj.IsDir() {
			continue
		}

		name := fObj.Name()
		if !ptrn.MatchString(name) {
			continue
		}

		ver, err := db.ParseVersion(name)
		if err != nil {
			return versions, err
		}

		versions = append(versions, ver)
	}

	sort.Slice(versions, func(i, j int) bool { return versions[i].Less(versions[j]) })

	return versions, nil
}

func listVersionFiles(path string, version db.Version) ([]string, []string) {
	ptrn := "*.sql"
	downSuffix := ".down.sql"

	files, err := filepath.Glob(filepath.Join(path, version.String(), ptrn))
	if err != nil {
		panic(err)
	}

	var upFiles, downFiles []string

	for _, f := range files {
		if strings.HasSuffix(f, downSuffix) {
			downFiles = append(downFiles, f)
			continue
		}

		upFiles = append(upFiles, f)
	}

	sort.Strings(upFiles)
	sort.Strings(downFiles)

	return upFiles, downFiles
}

func main() {
	if len(os.Args) < 3 || len(os.Args) > 4 {
		fmt.Println("Usage: migrate <db path> <migrations folder> <version>")
		os.Exit(ECodeBadArgs)
	}

	dbPath, migrPath := os.Args[1], os.Args[2]

	var targetVer db.Version

	if len(os.Args) == 4 {
		ver, err := db.ParseVersion(os.Args[3])
		if err != nil {
			fmt.Println("Usage: migrate <db path> <migrations folder> <version>")
			fmt.Println("Unable to parse target version:", err)
			os.Exit(ECodeBadArgs)
		}

		targetVer = ver
	}

	dbc, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		fmt.Println(err)
		os.Exit(ECodeBadDB)
	}

	defer dbc.Close()

	if err = db.PrepareVersionTable(dbc); err != nil {
		fmt.Println(err)
		os.Exit(ECodeBadDB)
	}

	dbVer, err := db.GetVersion(dbc)
	if err != nil {
		fmt.Println("Unable to get version from DB:", err)
		os.Exit(ECodeBadDB)
	}

	fmt.Println("DB version:", dbVer)

	versions, err := listVersions(migrPath)
	if err != nil {
		fmt.Println("Unable to get list of available versions:", err)
		os.Exit(ECodeBadPath)
	}

	if targetVer.IsZero() {
		targetVer = versions[len(versions)-1]
	} else {
		if !findVersion(targetVer, versions) {
			fmt.Println("Specified version wasn't found in list of available versions")
			os.Exit(ECodeBadArgs)
		}
	}

	if dbVer.Less(targetVer) {
		Upgrade(dbc, dbVer, targetVer, migrPath, versions)
	} else {
		Revert(dbc, dbVer, targetVer, migrPath, revertVersions(versions))
	}
}

func Upgrade(dbc *sql.DB, dbVer, targetVer db.Version, migrPath string, versions []db.Version) {
	for _, ver := range versions {
		if ver.Less(dbVer) {
			fmt.Println("\tSkipping version:", ver)
			continue
		}

		if !ver.Less(targetVer) { // reached next version after target, have no sense to continue
			fmt.Println("DB is upgraded")
			break
		}

		fmt.Println("\tApplying version:", ver)
		files, _ := listVersionFiles(migrPath, ver)

		err := db.ApplyVersion(dbc, ver, files)
		if err != nil {
			fmt.Println("Unable to upgrade DB:", err)
			os.Exit(ECodeMigrFailed)
		}
	}
}

func Revert(dbc *sql.DB, dbVer, targetVer db.Version, migrPath string, versions []db.Version) {
	for i, ver := range versions {
		if !ver.Less(dbVer) {
			fmt.Println("\tSkipping version:", ver)
			continue
		}

		if ver.Less(targetVer) { // reached target version, have no sense to continue
			fmt.Println("DB is reverted")
			break
		}

		fmt.Println("\tReverting version:", ver)
		_, files := listVersionFiles(migrPath, ver)

		err := db.ApplyVersion(dbc, versions[i+1], files)
		if err != nil {
			fmt.Println("Unable to upgrade DB:", err)
			os.Exit(ECodeMigrFailed)
		}
	}
}

func revertVersions(versions []db.Version) []db.Version {
	for i, j := 0, len(versions)-1; i < j; i, j = i+1, j-1 {
		versions[i], versions[j] = versions[j], versions[i]
	}

	return versions
}

func findVersion(version db.Version, versions []db.Version) bool {
	for _, ver := range versions {
		if ver.Equal(version) {
			return true
		}
	}

	return false
}
