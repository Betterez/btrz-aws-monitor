package betterauth

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"log"

	"github.com/mxk/go-sqlite/sqlite3"
)

// SQLiteAuthenticator - simple sqlite authenticator
type SQLiteAuthenticator struct {
	fileName         string
	sqliteConnection *sqlite3.Conn
	isOpen           bool
}

// GetUserLevel -gets user level
func (auth *SQLiteAuthenticator) GetUserLevel(username, password string) (int, error) {
	if !auth.isOpen {
		return 0, errors.New("Database connection is closed")
	}
	query := "select user_rank from users where username=$a and password_hash=$b"
	arguments := sqlite3.NamedArgs{"$a": username, "$b": fmt.Sprintf("%x", sha256.Sum256([]byte(password)))}
	row := make(sqlite3.RowMap)
	stt, err := auth.sqliteConnection.Query(query, arguments)
	if err != nil {
		log.Println(err, "can't get info")
		return 0, err
	}
	defer stt.Close()
	stt.Scan(row)
	if len(row) > 0 {
		userRank, _ := row["user_rank"].(int64)
		fmt.Println(userRank, arguments)
		return int(userRank), nil
	}
	return 0, nil
}

// GetSQLiteAuthenticator - return authenticator
func GetSQLiteAuthenticator(fileName string) (*SQLiteAuthenticator, error) {
	result := &SQLiteAuthenticator{fileName: fileName, isOpen: false}
	var err error
	result.sqliteConnection, err = sqlite3.Open(fileName)
	if err != nil {
		return nil, err
	}
	result.isOpen = true
	return result, nil
}
