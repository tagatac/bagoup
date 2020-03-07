package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/Masterminds/semver"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pkg/errors"
)

const _dbFileName = "chat.db"

var _reqOSVersion = semver.MustParse("10.13")

func main() {
	ok, err := validateOSVersion()
	if err != nil {
		log.Fatal(errors.Wrap(err, "validate OS version"))
	}
	if !ok {
		log.Fatalf("invalid OS version; update to Mac OS %s or newer", _reqOSVersion)
	}

	wd, err := os.Getwd()
	if err != nil {
		log.Fatal(errors.Wrap(err, "get working directory"))
	}
	dbPath := path.Join(wd, _dbFileName)
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatal(errors.Wrapf(err, "open DB file %q", dbPath))
	}
	defer db.Close()

	rows, err := db.Query("select distinct guid from chat")
	if err != nil {
		log.Fatal(errors.Wrap(err, "get chats by contact"))
	}
	defer rows.Close()
	for rows.Next() {
		var guid string
		err := rows.Scan(&guid)
		if err != nil {
			log.Fatal(errors.Wrap(err, "read guid"))
		}
		fmt.Printf("GUID: %s\n", guid)
		return
	}
}

func validateOSVersion() (bool, error) {
	cmd := exec.Command("sw_vers", "-productVersion")
	o, err := cmd.Output()
	if err != nil {
		return false, errors.Wrap(err, "call sw_vers")
	}
	vstr := strings.TrimSuffix(string(o), "\n")
	v, err := semver.NewVersion(vstr)
	if err != nil {
		return false, errors.Wrapf(err, "parse semantic version %q", vstr)
	}
	return !v.LessThan(_reqOSVersion), nil
}
