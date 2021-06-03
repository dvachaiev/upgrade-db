package main

import (
	"archive/tar"
	"compress/gzip"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/afero"
	"github.com/spf13/afero/tarfs"

	"github.com/dvachaiev/upgrade-db/db"
)

const (
	ECodeBadArgs = iota + 1
	ECodeBadDB
	ECodeBadPath
	ECodeMigrFailed
)

func usageError(msg string) {
	fmt.Printf("Usage: %s <db path> <migrations location> [<version>]\n", os.Args[0])

	if msg != "" {
		fmt.Println(msg)
	}

	os.Exit(ECodeBadArgs)
}

func listVersions(fs afero.Fs) ([]db.Version, error) {
	ptrn := regexp.MustCompile(`^[0-9]+(\.[0-9]+)*$`)

	var versions []db.Version

	files, err := afero.ReadDir(fs, ".")
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

func listVersionFiles(fs afero.Fs, version db.Version) ([]string, []string) {
	ptrn := "*.sql"
	downSuffix := ".down.sql"

	files, err := afero.Glob(fs, filepath.Join(version.String(), ptrn))
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

	for i, j := 0, len(downFiles)-1; i < j; i, j = i+1, j-1 {
		downFiles[i], downFiles[j] = downFiles[j], downFiles[i]
	}
	return upFiles, downFiles
}

func main() {
	if len(os.Args) < 3 || len(os.Args) > 4 {
		usageError("")
	}

	dbPath, migrPath := os.Args[1], os.Args[2]

	fs, err := getFS(migrPath)
	if err != nil {
		usageError(fmt.Sprintf("Migrations path error: %s", err))
	}

	var targetVer db.Version

	if len(os.Args) == 4 {
		ver, err := db.ParseVersion(os.Args[3])
		if err != nil {
			usageError(fmt.Sprintf("Unable to parse target version: %s", err))
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

	versions, err := listVersions(fs)
	if err != nil {
		fmt.Println("Unable to get list of available versions:", err)
		os.Exit(ECodeBadPath)
	}

	if targetVer.IsZero() {
		targetVer = versions[len(versions)-1]
	}

	fmt.Println("Target version:", targetVer)

	if dbVer.Less(targetVer) {
		Upgrade(dbc, dbVer, targetVer, fs, versions)
	} else {
		Revert(dbc, dbVer, targetVer, fs, revertVersions(versions))
	}
}

func Upgrade(dbc *sql.DB, dbVer, targetVer db.Version, fs afero.Fs, versions []db.Version) {
	for _, ver := range versions {
		if ver.Less(dbVer) {
			fmt.Println("\tSkipping version:", ver)
			continue
		}

		if !ver.Less(targetVer) { // reached next version after target, have no sense to continue
			break
		}

		fmt.Println("\tApplying version:", ver)
		files, _ := listVersionFiles(fs, ver)

		err := db.ApplyVersion(dbc, ver, fs, files)
		if err != nil {
			fmt.Println("Unable to upgrade DB:", err)
			os.Exit(ECodeMigrFailed)
		}
	}
}

func Revert(dbc *sql.DB, dbVer, targetVer db.Version, fs afero.Fs, versions []db.Version) {
	for i, ver := range versions {
		if !ver.Less(dbVer) {
			fmt.Println("\tSkipping version:", ver)
			continue
		}

		if ver.Less(targetVer) { // reached target version, have no sense to continue
			break
		}

		fmt.Println("\tReverting version:", ver)
		_, files := listVersionFiles(fs, ver)

		var nextVer db.Version

		if i+1 < len(versions) {
			nextVer = versions[i+1]
		} else {
			nextVer = targetVer // set target version if need roll back all changes
		}

		err := db.ApplyVersion(dbc, nextVer, fs, files)
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

func getFS(path string) (afero.Fs, error) {
	stat, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	if stat.IsDir() {
		return afero.NewBasePathFs(afero.NewOsFs(), path), nil
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	var r io.Reader = f

	switch {
	case strings.HasSuffix(path, ".tgz") || strings.HasSuffix(path, ".tar.gz"):
		gr, err := gzip.NewReader(r)
		if err != nil {
			return nil, err
		}

		r = gr

		fallthrough
	case strings.HasSuffix(path, ".tar"):

		return tarfs.New(tar.NewReader(r)), nil
	}

	return nil, errors.New("unsupported migrations path")
}
