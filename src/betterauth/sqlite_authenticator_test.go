package betterauth

import (
	"crypto/sha256"
	"fmt"
	"testing"

	sqlite3 "github.com/mxk/go-sqlite/sqlite3"
)

func TestAuthentication(t *testing.T) {
	expectedPassword := "123456"
	db, err := sqlite3.Open("../../test-objects/users.sqlite")
	if err != nil {
		t.Fatalf("db error %v opening db", err)
	}
	defer db.Close()
	query := "select * from users where username='tal'"

	row := make(sqlite3.RowMap)
	stt, err := db.Query(query)
	if err != nil {
		t.Fatalf("db error %v", err)
	}
	stt.Scan(row)
	defer stt.Close()
	var sum string = fmt.Sprintf("%x", sha256.Sum256([]byte(expectedPassword)))
	if row["password_hash"] != sum {
		t.Fatalf("Bad value found for hash:%s", sum)
	}

}

func TestAuthenticationWithParams(t *testing.T) {
	expectedPassword := "123456"
	db, err := sqlite3.Open("../../test-objects/users.sqlite")
	if err != nil {
		t.Fatalf("db error %v opening db", err)
	}
	defer db.Close()
	query := "select * from users where username=$a"
	arguments := sqlite3.NamedArgs{"$a": "tal"}
	row := make(sqlite3.RowMap)
	stt, err := db.Query(query, arguments)
	if err != nil {
		t.Fatalf("db error %v", err)
	}
	stt.Scan(row)
	defer stt.Close()
	var sum string = fmt.Sprintf("%x", sha256.Sum256([]byte(expectedPassword)))
	if row["password_hash"] != sum {
		t.Fatalf("Bad value found for hash:%s", sum)
	}

}

func TestParamsSanitation(t *testing.T) {
	expectedPassword := "123456"
	db, err := sqlite3.Open("../../test-objects/users.sqlite")
	if err != nil {
		t.Fatalf("db error %v opening db", err)
	}
	defer db.Close()
	query := "select * from users where username=$a"
	arguments := sqlite3.NamedArgs{"$a": "tamtam;DROP TABLE users"}
	db.Query(query, arguments)
	arguments = sqlite3.NamedArgs{"$a": "tal"}
	stt, err := db.Query(query, arguments)
	if err != nil {
		t.Fatalf("db error %v", err)
	}
	defer stt.Close()
	row := make(sqlite3.RowMap)
	stt.Scan(row)
	var sum string = fmt.Sprintf("%x", sha256.Sum256([]byte(expectedPassword)))
	if row["password_hash"] != sum {
		t.Fatalf("Bad value found for hash:%s", sum)
	}
}
