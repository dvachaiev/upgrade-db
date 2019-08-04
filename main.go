package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"

	"github.com/dvachaiev/upgrade-db/db"
	_ "github.com/mattn/go-sqlite3"
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

func listVersionFiles(path string, version db.Version) []string {
	ptrn := "*.sql"
	files, err := filepath.Glob(filepath.Join(path, version.String(), ptrn))
	if err != nil {
		panic(err)
	}
	sort.Strings(files)
	return files
}

func main() {
	if len(os.Args) != 3 {
		fmt.Println("Usage: migrate <db path> <migrations folder>")
		os.Exit(1)
	}
	dbPath, migrPath := os.Args[1], os.Args[2]

	dbc, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		fmt.Println(err)
		os.Exit(2)
	}
	defer dbc.Close()

	if err = db.CreateVersionTable(dbc); err != nil {
		fmt.Println(err)
		os.Exit(2)
	}

	dbVer, err := db.GetVersion(dbc)
	if err != nil {
		fmt.Println("Unable to get version from DB:", err)
		os.Exit(2)
	}
	fmt.Println("DB version:", dbVer)

	versions, err := listVersions(migrPath)
	if err != nil {
		fmt.Println("Unable to get list of available versions:", err)
		os.Exit(3)
	}
	for _, ver := range versions {
		if ver.Less(dbVer) {
			fmt.Println("\tSkipping version:", ver)
			continue
		}
		fmt.Println("\tApplying version:", ver)
		files := listVersionFiles(migrPath, ver)
		err = db.UpgradeVersion(dbc, ver, files)
		if err != nil {
			fmt.Println("Unable to upgrade DB:", err)
			os.Exit(4)
		}
	}

}
