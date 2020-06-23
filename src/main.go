package main

import (
	"bufio"
	"database/sql"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/robfig/cron"
	"github.com/spf13/viper"
	"log"
	"os"
)

func migrate() {
	db2, err := sql.Open("mysql", viper.GetString("sql_source"))
	fatalErrorHandle(&err, "error opening sql db")
	rows, err2 := db2.Query("SELECT email_hash FROM user_info UNION SELECT email_hash FROM verification_codes")
	fatalErrorHandle(&err2, "failed step 1!")

	var emailHash, primKey string
	hashMap := make(map[string]string)
	for rows.Next() {
		err2 = rows.Scan(&emailHash)
		fatalErrorHandle(&err2, "failed step 2.1!")
		hashMap[emailHash] = hash1(viper.GetString("salt") + emailHash)
	}

	err2 = rows.Err()
	if err2 != nil {
		fatalErrorHandle(&err2, "failed step 2.2!")
	}

	migrateTable := func(table string, primKeyName string) {
		rows3, err3 := db2.Query("SELECT email_hash, " + primKeyName + " FROM " + table)
		fatalErrorHandle(&err3, "failed step 3.1! table="+table)
		for rows3.Next() {
			err3 = rows.Scan(&emailHash, &primKey)
			fatalErrorHandle(&err3, "failed step 3.2! table="+table)
			smt, err4 := db2.Prepare("UPDATE " + table + " SET email_hash=? WHERE " + primKeyName + "=?")
			fatalErrorHandle(&err4, "failed step 3.3! table="+table)
			_, err5 := smt.Exec(hashMap[emailHash], primKey)
			fatalErrorHandle(&err5, "failed step 3.4! table="+table)
		}
		fmt.Println("finished migrate table=" + table)
	}
	migrateTable("user_info", "email_hash")
	migrateTable("verification_codes", "email_hash")
	migrateTable("posts", "pid")
	migrateTable("comments", "cid")
	migrateTable("reports", "rid")
	migrateTable("banned", "email_hash")
}

func main() {
	initLog()
	initConfigFile()
	initDb()

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("begin migration?(y/n): ")
	text, _ := reader.ReadString('\n')
	if text == "y" {
		fmt.Println("begin migration...")
		migrate()
	}

	log.Println("start timestamp: ", getTimeStamp())
	if false == viper.GetBool("is_debug") {
		gin.SetMode(gin.ReleaseMode)
	}

	var err error
	hotPosts, _ = dbGetHotPosts()
	c := cron.New()
	_, _ = c.AddFunc("*/10 * * * *", func() {
		hotPosts, err = dbGetHotPosts()
		log.Println("refreshed hotPosts ,err=", err)
	})
	c.Start()

	listenHttp()

}
