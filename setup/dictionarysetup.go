package main

import (
	"database/sql"
	"os"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	db, _ := sql.Open("sqlite3", "words.db")
	content, _ := os.ReadFile("words.txt")
	lines := strings.Split(string(content), "\n")

	_, err := db.Exec("CREATE TABLE words (word TEXT);")
	if err != nil {
		panic("something is wrong with sqlite")
	}
	queryStr := "insert into words (word) values "
	builder := strings.Builder{}
	builder.WriteString(queryStr)
	for _, line := range lines {
		line = strings.Replace(line, "\r", "", -1)
		if strings.Contains(line, "'") {
			continue
		}
		line = strings.ToLower(line)
		builder.WriteString("('" + line + "'),")
	}
	queryStr = builder.String()
	_, err = db.Exec(queryStr[:len(queryStr)-1])
	if err != nil {
		panic("something is wrong with sqlite" + err.Error())
	}
}
