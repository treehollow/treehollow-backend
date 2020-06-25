package main

import (
	"database/sql"
	"fmt"
	"github.com/spf13/viper"
	"strconv"
	"strings"
)

func migrate() {
	db2, err := sql.Open("mysql", viper.GetString("sql_source"))
	fatalErrorHandle(&err, "error opening sql db")
	rows, err2 := db2.Query("SELECT email_hash, attentions FROM user_info")
	fatalErrorHandle(&err2, "failed step 1.1!")
	maxPid, err3 := dbGetMaxPid()
	fatalErrorHandle(&err3, "failed step 1.2!")
	_, err5 := db.Exec("create table attentions\n(\n    email_hash CHAR(64) NOT NULL,\n    pid        INT      NOT NULL,\n    INDEX (email_hash),\n    CONSTRAINT pid_email UNIQUE (pid, email_hash)\n) DEFAULT CHARSET = ascii")
	fatalErrorHandle(&err5, "failed step 1.3!")

	var emailHash, s string
	var inserts []string
	{
	}
	for rows.Next() {
		err2 = rows.Scan(&emailHash, &s)
		fatalErrorHandle(&err2, "failed step 2.1!")
		pids := hexToIntSlice(s)
		inserts = []string{}
		for _, pid := range pids {
			if pid <= maxPid {
				inserts = append(inserts, "('"+emailHash+"',"+strconv.Itoa(pid)+")")
			}
		}
		if len(inserts) > 0 {
			query := "INSERT INTO attentions(email_hash, pid) VALUES " + strings.Join(inserts, ",")
			_, err4 := db2.Exec(query)
			fatalErrorHandle(&err4, "failed step 2.2!")
		}
	}
	fmt.Println("migrate success!")
}

func main() {
	initLog()
	initConfigFile()
	initDb()

	migrate()

	//log.Println("start timestamp: ", getTimeStamp())
	//if false == viper.GetBool("is_debug") {
	//	gin.SetMode(gin.ReleaseMode)
	//}
	//
	//var err error
	//hotPosts, _ = dbGetHotPosts()
	//c := cron.New()
	//_, _ = c.AddFunc("*/1 * * * *", func() {
	//	hotPosts, err = dbGetHotPosts()
	//	log.Println("refreshed hotPosts ,err=", err)
	//})
	//c.Start()
	//
	//listenHttp()

}
