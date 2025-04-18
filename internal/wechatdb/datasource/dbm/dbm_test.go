package dbm

import (
	"fmt"
	"testing"
	"time"
)

func TestXxx(t *testing.T) {
	path := "/Users/sarv/Documents/chatlog/bigjun_9e7a"

	g := &Group{
		Name:      "session",
		Pattern:   `session\.db$`,
		BlackList: []string{},
	}

	d := NewDBManager(path)
	d.AddGroup(g)
	d.Start()

	i := 0
	for {
		db, err := d.GetDB("session")
		if err != nil {
			fmt.Println(err)
			break
		}

		var username string
		row := db.QueryRow(`SELECT username FROM SessionTable LIMIT 1`)
		if err := row.Scan(&username); err != nil {
			fmt.Printf("Error scanning row: %v\n", err)
			time.Sleep(100 * time.Millisecond)
			continue
		}
		fmt.Printf("%d: Username: %s\n", i, username)
		i++
		time.Sleep(1000 * time.Millisecond)
	}

}
