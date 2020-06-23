package main

import (
	"database/sql"
	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"github.com/spf13/viper"
	"log"
)

var db *sql.DB
var saveCodeIns *sql.Stmt
var checkCodeOut *sql.Stmt
var saveTokenIns *sql.Stmt
var doPostIns *sql.Stmt
var getInfoOut *sql.Stmt
var setAttentionsIns *sql.Stmt
var doCommentIns *sql.Stmt
var doReportIns *sql.Stmt
var checkCommentNameOut *sql.Stmt
var getCommentCountOut *sql.Stmt
var plusOneCommentIns *sql.Stmt
var plusOneReportIns *sql.Stmt
var plus666ReportIns *sql.Stmt
var plusOneAttentionIns *sql.Stmt
var minusOneAttentionIns *sql.Stmt
var getOnePostOut *sql.Stmt
var getCommentsOut *sql.Stmt
var getPostsOut *sql.Stmt
var searchOut *sql.Stmt
var hotPostsOut *sql.Stmt
var bannedTimesOut *sql.Stmt
var banIns *sql.Stmt
var getBannedOut *sql.Stmt

func initDb() {
	var err error
	db, err = sql.Open("mysql", viper.GetString("sql_source"))
	fatalErrorHandle(&err, "error opening sql db")

	saveCodeIns, err = db.Prepare("INSERT INTO verification_codes (email_hash, timestamp, code) VALUES (?, ?, ?) ON DUPLICATE KEY UPDATE timestamp=?, code=?")
	fatalErrorHandle(&err, "error preparing verification_codes sql query")

	checkCodeOut, err = db.Prepare("SELECT timestamp, code FROM verification_codes WHERE email_hash=?")
	fatalErrorHandle(&err, "error preparing verification_codes sql query")

	saveTokenIns, err = db.Prepare("INSERT INTO user_info (email_hash, token, timestamp, attentions) VALUES (?, ?, ?, ?) ON DUPLICATE KEY UPDATE timestamp=?, token=?")
	fatalErrorHandle(&err, "error preparing user_info sql query")

	doPostIns, err = db.Prepare("INSERT INTO posts (email_hash, text, timestamp, tag, type ,file_path, likenum, replynum, reportnum) VALUES ((SELECT email_hash FROM user_info WHERE token=?), ?, ?, ?, ?, ?, 0, 0, 0)")
	fatalErrorHandle(&err, "error preparing posts sql query")

	getInfoOut, err = db.Prepare("SELECT attentions, email_hash FROM user_info WHERE token=?")
	fatalErrorHandle(&err, "error preparing user_info sql query")

	setAttentionsIns, err = db.Prepare("UPDATE user_info SET attentions=? WHERE token=?")
	fatalErrorHandle(&err, "error preparing user_info sql query")

	doCommentIns, err = db.Prepare("INSERT INTO comments (email_hash, pid, text, tag, timestamp, name) VALUES ((SELECT email_hash FROM user_info WHERE token=?), ?, ?, ?, ?, ?)")
	fatalErrorHandle(&err, "error preparing comments sql query")

	doReportIns, err = db.Prepare("INSERT INTO reports (email_hash, pid, reason, timestamp) VALUES ((SELECT email_hash FROM user_info WHERE token=?), ?, ?, ?)")
	fatalErrorHandle(&err, "error preparing reports sql query")

	checkCommentNameOut, err = db.Prepare("SELECT name FROM comments WHERE pid=? AND email_hash=(SELECT email_hash FROM user_info WHERE token=?)")
	fatalErrorHandle(&err, "error preparing comments sql query")

	getCommentCountOut, err = db.Prepare("SELECT count( DISTINCT(email_hash) ) FROM comments WHERE pid=? AND email_hash != ?")
	fatalErrorHandle(&err, "error preparing comments sql query")

	getOnePostOut, err = db.Prepare("SELECT email_hash, text, timestamp, tag, type, file_path, likenum, replynum, reportnum FROM posts WHERE pid=? AND reportnum<10")
	fatalErrorHandle(&err, "error preparing posts sql query")

	plusOneCommentIns, err = db.Prepare("UPDATE posts SET replynum=replynum+1 WHERE pid=?")
	fatalErrorHandle(&err, "error preparing posts sql query")

	plusOneReportIns, err = db.Prepare("UPDATE posts SET reportnum=reportnum+1 WHERE pid=?")
	fatalErrorHandle(&err, "error preparing posts sql query")

	plus666ReportIns, err = db.Prepare("UPDATE posts SET reportnum=reportnum+666 WHERE pid=?")
	fatalErrorHandle(&err, "error preparing posts sql query")

	plusOneAttentionIns, err = db.Prepare("UPDATE posts SET likenum=likenum+1 WHERE pid=?")
	fatalErrorHandle(&err, "error preparing posts sql query")

	minusOneAttentionIns, err = db.Prepare("UPDATE posts SET likenum=likenum-1 WHERE pid=?")
	fatalErrorHandle(&err, "error preparing posts sql query")

	getCommentsOut, err = db.Prepare("SELECT cid, email_hash, text, tag, timestamp, name FROM comments WHERE pid=?")
	fatalErrorHandle(&err, "error preparing comments sql query")

	getPostsOut, err = db.Prepare("SELECT pid, email_hash, text, timestamp, tag, type, file_path, likenum, replynum FROM posts WHERE pid>? AND pid<=? AND reportnum<10 ORDER BY pid DESC")
	fatalErrorHandle(&err, "error preparing posts sql query")

	searchOut, err = db.Prepare("SELECT pid, email_hash, text, timestamp, tag, type, file_path, likenum, replynum FROM posts WHERE match(text) against(? IN BOOLEAN MODE) AND reportnum<10 ORDER BY pid DESC LIMIT ?, ?")
	fatalErrorHandle(&err, "error preparing posts sql query")

	hotPostsOut, err = db.Prepare("SELECT pid, email_hash, text, timestamp, tag, type, file_path, likenum, replynum FROM posts WHERE pid>(SELECT MAX(pid)-1000 FROM posts) AND reportnum<10 ORDER BY replynum DESC")
	fatalErrorHandle(&err, "error preparing posts sql query")

	bannedTimesOut, err = db.Prepare("SELECT COUNT(*) FROM banned WHERE email_hash=? AND expire_time>?")
	fatalErrorHandle(&err, "error preparing banned sql query")

	banIns, err = db.Prepare("INSERT INTO banned (email_hash, reason, timestamp, expire_time) VALUES (?, ?, ?, ?)")
	fatalErrorHandle(&err, "error preparing banned sql query")

	getBannedOut, err = db.Prepare("SELECT reason, timestamp, expire_time FROM banned WHERE email_hash=? ORDER BY timestamp DESC")
	fatalErrorHandle(&err, "error preparing banned sql query")
}

func dbGetOnePost(pid int) (string, string, int, string, string, string, int, int, int, error) {
	var emailHash, text, tag, typ, filePath string
	var timestamp, likenum, replynum, reportnum int
	err := getOnePostOut.QueryRow(pid).Scan(&emailHash, &text, &timestamp, &tag, &typ, &filePath, &likenum, &replynum, &reportnum)
	return emailHash, text, timestamp, tag, typ, filePath, likenum, replynum, reportnum, err
}

func dbBannedTimesPost(dzEmailHash string, fromTimestamp int) (int, error) {
	var bannedTimes int
	err := bannedTimesOut.QueryRow(dzEmailHash, fromTimestamp).Scan(&bannedTimes)
	return bannedTimes, err
}

func dbSaveBanUser(dzEmailHash string, reason string, interval int) error {
	timestamp := int(getTimeStamp())
	_, err := banIns.Exec(dzEmailHash, reason, timestamp, timestamp+interval)

	return err
}

func dbGetPostsByPidList(pids []int) ([]interface{}, error) {
	var rtn []interface{}
	rows, err := db.Query("SELECT pid, email_hash, text, timestamp, tag, type, file_path, likenum, replynum FROM posts WHERE pid IN (" + SplitToString(pids, ",") + ") AND reportnum<10 ORDER BY pid DESC")
	if err != nil {
		return nil, err
	}

	var emailHash, text, tag, typ, filePath string
	var timestamp, pid, likenum, replynum int
	for rows.Next() {
		err := rows.Scan(&pid, &emailHash, &text, &timestamp, &tag, &typ, &filePath, &likenum, &replynum)
		if err != nil {
			log.Fatal(err)
		}
		rtn = append(rtn, gin.H{
			"pid":       pid,
			"text":      text,
			"type":      typ,
			"timestamp": timestamp,
			"reply":     replynum,
			"likenum":   likenum,
			"url":       filePath,
			"tag":       IfThenElse(len(tag) != 0, tag, nil),
		})
	}
	err = rows.Err()
	if err != nil {
		return nil, err
	}
	return rtn, nil
}

func dbGetSavedPosts(pidMin int, pidMax int) ([]interface{}, error) {
	var rtn []interface{}
	rows, err := getPostsOut.Query(pidMin, pidMax)
	if err != nil {
		return nil, err
	}

	var emailHash, text, tag, typ, filePath string
	var timestamp, pid, likenum, replynum int
	pinnedPids := getPinnedPids()
	for rows.Next() {
		err := rows.Scan(&pid, &emailHash, &text, &timestamp, &tag, &typ, &filePath, &likenum, &replynum)
		if err != nil {
			log.Fatal(err)
		}
		if _, ok := containsInt(pinnedPids, pid); !ok {
			rtn = append(rtn, gin.H{
				"pid":       pid,
				"text":      text,
				"type":      typ,
				"timestamp": timestamp,
				"reply":     replynum,
				"likenum":   likenum,
				"url":       filePath,
				"tag":       IfThenElse(len(tag) != 0, tag, nil),
			})
		}
	}
	err = rows.Err()
	if err != nil {
		return nil, err
	}
	return rtn, nil
}

func dbGetBannedMsgs(emailHash string) ([]interface{}, error) {
	var rtn []interface{}
	rows, err := getBannedOut.Query(emailHash)
	if err != nil {
		return nil, err
	}

	var reason string
	var timestamp, expireTime int
	for rows.Next() {
		err := rows.Scan(&reason, &timestamp, &expireTime)
		if err != nil {
			log.Fatal(err)
		}
		rtn = append(rtn, gin.H{
			"content":   reason,
			"timestamp": timestamp,
			"title":     "提示",
		})
	}
	err = rows.Err()
	if err != nil {
		return nil, err
	}
	if len(rtn) == 0 {
		rtn = append(rtn, gin.H{
			"content":   "目前尚无系统消息",
			"timestamp": 0,
			"title":     "提示",
		})
	}
	return rtn, nil
}

func dbSearchSavedPosts(str string, limitMin int, searchPageSize int) ([]interface{}, error) {
	var rtn []interface{}
	rows, err := searchOut.Query(str, limitMin, searchPageSize)
	if err != nil {
		return nil, err
	}

	var emailHash, text, tag, typ, filePath string
	var timestamp, pid, likenum, replynum int
	for rows.Next() {
		err := rows.Scan(&pid, &emailHash, &text, &timestamp, &tag, &typ, &filePath, &likenum, &replynum)
		if err != nil {
			log.Fatal(err)
		}
		rtn = append(rtn, gin.H{
			"pid":       pid,
			"text":      text,
			"type":      typ,
			"timestamp": timestamp,
			"reply":     replynum,
			"likenum":   likenum,
			"url":       filePath,
			"tag":       IfThenElse(len(tag) != 0, tag, nil),
		})
	}
	err = rows.Err()
	if err != nil {
		return nil, err
	}
	return rtn, nil
}

func dbGetHotPosts() ([]interface{}, error) {
	var rtn []interface{}
	rows, err := hotPostsOut.Query()
	if err != nil {
		return nil, err
	}

	var emailHash, text, tag, typ, filePath string
	var timestamp, pid, likenum, replynum int
	for rows.Next() {
		err := rows.Scan(&pid, &emailHash, &text, &timestamp, &tag, &typ, &filePath, &likenum, &replynum)
		if err != nil {
			log.Fatal(err)
		}
		rtn = append(rtn, gin.H{
			"pid":       pid,
			"text":      text,
			"type":      typ,
			"timestamp": timestamp,
			"reply":     replynum,
			"likenum":   likenum,
			"url":       filePath,
			"tag":       IfThenElse(len(tag) != 0, tag, nil),
		})
	}
	err = rows.Err()
	if err != nil {
		return nil, err
	}
	return rtn, nil
}

func dbGetSavedComments(pid int) ([]interface{}, error) {
	var rtn []interface{}
	rows, err := getCommentsOut.Query(pid)
	if err != nil {
		return nil, err
	}

	var text, tag, name, emailHash string
	var cid, timestamp int
	for rows.Next() {
		err := rows.Scan(&cid, &emailHash, &text, &tag, &timestamp, &name)
		if err != nil {
			log.Fatal(err)
		}
		rtn = append(rtn, gin.H{
			"cid":       cid,
			"pid":       pid,
			"text":      "[" + name + "] " + text,
			"timestamp": timestamp,
			"tag":       IfThenElse(len(tag) != 0, tag, nil),
			"name":      name,
		})
	}
	err = rows.Err()
	if err != nil {
		return nil, err
	}
	return rtn, nil
}

func dbSaveCode(user string, code string) error {
	timestamp := int32(getTimeStamp())
	_, err := saveCodeIns.Exec(hashEmail(user), timestamp, code, timestamp, code)

	return err
}

func dbGetCode(hashedUser string) (string, int64, error) {
	var timestamp int64
	var correctCode string
	err := checkCodeOut.QueryRow(hashedUser).Scan(&timestamp, &correctCode)
	if err != nil {
		return "", -1, err
	}
	return correctCode, timestamp, nil
}

func dbSaveToken(token string, hashedUser string) error {
	timestamp := int32(getTimeStamp())
	_, err := saveTokenIns.Exec(hashedUser, token, timestamp, "", timestamp, token)
	return err
}

func dbGetCommentNameByToken(token string, pid int) (string, error) {
	var name string
	err := checkCommentNameOut.QueryRow(pid, token).Scan(&name)
	return name, err
}

func dbGetMaxPid() (int, error) {
	var pid int64
	err := db.QueryRow("SELECT MAX(pid) FROM posts").Scan(&pid)
	return int(pid), err
}

func dbGetCommentCount(pid int, dzEmailHash string) (int, error) {
	var rtn int64
	err := getCommentCountOut.QueryRow(pid, dzEmailHash).Scan(&rtn)
	return int(rtn), err
}

func dbSavePost(token string, text string, tag string, typ string, filePath string) (int, error) {
	timestamp := int32(getTimeStamp())
	res, err := doPostIns.Exec(token, text, timestamp, tag, typ, filePath)
	if err != nil {
		return -1, err
	}
	var id int64
	id, err = res.LastInsertId()
	if err != nil {
		return -1, err
	} else {
		return int(id), nil
	}
}

func dbSaveComment(token string, text string, tag string, pid int, name string) (int, error) {
	timestamp := int32(getTimeStamp())
	res, err := doCommentIns.Exec(token, pid, text, tag, timestamp, name)
	if err != nil {
		return -1, err
	}
	var id int64
	id, err = res.LastInsertId()
	if err != nil {
		return -1, err
	} else {
		return int(id), nil
	}
}

func dbSaveReport(token string, reason string, pid int) (int, error) {
	timestamp := int32(getTimeStamp())
	res, err := doReportIns.Exec(token, pid, reason, timestamp)
	if err != nil {
		return -1, err
	}
	var id int64
	id, err = res.LastInsertId()
	if err != nil {
		return -1, err
	} else {
		return int(id), nil
	}
}

func dbGetInfoByToken(token string) (string, string, error) {
	var attentions, emailHash string
	err := getInfoOut.QueryRow(token).Scan(&attentions, &emailHash)
	return attentions, emailHash, err
}

func dbSetAttentions(token string, attentions string) (bool, error) {
	res, err := setAttentionsIns.Exec(attentions, token)
	if err != nil {
		return false, err
	}
	var affected int64
	affected, err = res.RowsAffected()
	if err != nil {
		return false, err
	}
	return affected != 0, nil
}
