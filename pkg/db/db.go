package db

import (
	"database/sql"
	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"github.com/spf13/viper"
	"log"
	"thuhole-go-backend/pkg/utils"
)

var db *sql.DB
var saveCodeIns *sql.Stmt
var checkCodeOut *sql.Stmt
var saveTokenIns *sql.Stmt
var doPostIns *sql.Stmt
var getInfoOut *sql.Stmt
var doCommentIns *sql.Stmt
var doReportIns *sql.Stmt
var checkCommentNameOut *sql.Stmt
var getCommentCountOut *sql.Stmt
var PlusOneCommentIns *sql.Stmt
var PlusOneReportIns *sql.Stmt
var Plus666ReportIns *sql.Stmt
var PlusOneAttentionIns *sql.Stmt
var MinusOneAttentionIns *sql.Stmt
var getOnePostOut *sql.Stmt
var getCommentsOut *sql.Stmt
var getPostsOut *sql.Stmt
var getAttentionPidsOut *sql.Stmt
var AddAttentionIns *sql.Stmt
var RemoveAttentionIns *sql.Stmt
var isAttentionOut *sql.Stmt
var searchOut *sql.Stmt
var deletedOut *sql.Stmt
var hotPostsOut *sql.Stmt
var bannedTimesOut *sql.Stmt
var banIns *sql.Stmt
var getBannedOut *sql.Stmt
var SetPostTagIns *sql.Stmt
var SetCommentTagIns *sql.Stmt
var reportsOut *sql.Stmt
var bansOut *sql.Stmt

func InitDb() {
	var err error
	db, err = sql.Open("mysql", viper.GetString("sql_source"))
	utils.FatalErrorHandle(&err, "error opening sql db")

	//VERIFICATION CODES
	saveCodeIns, err = db.Prepare("INSERT INTO verification_codes (email_hash, timestamp, code) VALUES (?, ?, ?) ON DUPLICATE KEY UPDATE timestamp=?, code=?")
	utils.FatalErrorHandle(&err, "error preparing verification_codes sql query")

	checkCodeOut, err = db.Prepare("SELECT timestamp, code FROM verification_codes WHERE email_hash=?")
	utils.FatalErrorHandle(&err, "error preparing verification_codes sql query")

	//USER INFO
	saveTokenIns, err = db.Prepare("INSERT INTO user_info (email_hash, token, timestamp) VALUES (?, ?, ?) ON DUPLICATE KEY UPDATE timestamp=?, token=?")
	utils.FatalErrorHandle(&err, "error preparing user_info sql query")

	getInfoOut, err = db.Prepare("SELECT email_hash FROM user_info WHERE token=?")
	utils.FatalErrorHandle(&err, "error preparing user_info sql query")

	//POSTS
	doPostIns, err = db.Prepare("INSERT INTO posts (email_hash, text, timestamp, tag, type ,file_path, likenum, replynum, reportnum) VALUES (?, ?, ?, ?, ?, ?, 0, 0, 0)")
	utils.FatalErrorHandle(&err, "error preparing posts sql query")

	getOnePostOut, err = db.Prepare("SELECT email_hash, text, timestamp, tag, type, file_path, likenum, replynum, reportnum FROM posts WHERE pid=? AND reportnum<10")
	utils.FatalErrorHandle(&err, "error preparing posts sql query")

	PlusOneCommentIns, err = db.Prepare("UPDATE posts SET replynum=replynum+1 WHERE pid=?")
	utils.FatalErrorHandle(&err, "error preparing posts sql query")

	PlusOneReportIns, err = db.Prepare("UPDATE posts SET reportnum=reportnum+1 WHERE pid=?")
	utils.FatalErrorHandle(&err, "error preparing posts sql query")

	Plus666ReportIns, err = db.Prepare("UPDATE posts SET reportnum=reportnum+666 WHERE pid=?")
	utils.FatalErrorHandle(&err, "error preparing posts sql query")

	PlusOneAttentionIns, err = db.Prepare("UPDATE posts SET likenum=likenum+1 WHERE pid=?")
	utils.FatalErrorHandle(&err, "error preparing posts sql query")

	MinusOneAttentionIns, err = db.Prepare("UPDATE posts SET likenum=likenum-1 WHERE pid=?")
	utils.FatalErrorHandle(&err, "error preparing posts sql query")

	SetPostTagIns, err = db.Prepare("UPDATE posts SET tag=? WHERE pid=?")
	utils.FatalErrorHandle(&err, "error preparing posts sql query")

	getPostsOut, err = db.Prepare("SELECT pid, email_hash, text, timestamp, tag, type, file_path, likenum, replynum, reportnum FROM posts WHERE pid>? AND pid<=? AND reportnum<10 ORDER BY pid DESC")
	utils.FatalErrorHandle(&err, "error preparing posts sql query")

	searchOut, err = db.Prepare("SELECT pid, email_hash, text, timestamp, tag, type, file_path, likenum, replynum, reportnum FROM posts WHERE match(text) against(? IN BOOLEAN MODE) AND reportnum<10 ORDER BY pid DESC LIMIT ?, ?")
	utils.FatalErrorHandle(&err, "error preparing posts sql query")

	hotPostsOut, err = db.Prepare("SELECT pid, email_hash, text, timestamp, tag, type, file_path, likenum, replynum, reportnum FROM posts WHERE pid>(SELECT MAX(pid)-1000 FROM posts) AND reportnum<10 ORDER BY likenum*3+replynum+timestamp/900-reportnum*10 DESC")
	utils.FatalErrorHandle(&err, "error preparing posts sql query")

	deletedOut, err = db.Prepare("SELECT pid, email_hash, text, timestamp, tag, type, file_path, likenum, replynum, reportnum FROM posts WHERE reportnum>=10 ORDER BY pid DESC LIMIT ?, ?")
	utils.FatalErrorHandle(&err, "error preparing posts sql query")

	//COMMENTS
	getCommentsOut, err = db.Prepare("SELECT cid, email_hash, text, tag, timestamp, name FROM comments WHERE pid=?")
	utils.FatalErrorHandle(&err, "error preparing comments sql query")

	doCommentIns, err = db.Prepare("INSERT INTO comments (email_hash, pid, text, tag, timestamp, name) VALUES (?, ?, ?, ?, ?, ?)")
	utils.FatalErrorHandle(&err, "error preparing comments sql query")

	checkCommentNameOut, err = db.Prepare("SELECT name FROM comments WHERE pid=? AND email_hash=?")
	utils.FatalErrorHandle(&err, "error preparing comments sql query")

	getCommentCountOut, err = db.Prepare("SELECT count( DISTINCT(email_hash) ) FROM comments WHERE pid=? AND email_hash != ?")
	utils.FatalErrorHandle(&err, "error preparing comments sql query")

	SetCommentTagIns, err = db.Prepare("UPDATE comments SET tag=? WHERE cid=?")
	utils.FatalErrorHandle(&err, "error preparing comments sql query")

	//REPORTS
	doReportIns, err = db.Prepare("INSERT INTO reports (email_hash, pid, reason, timestamp) VALUES (?, ?, ?, ?)")
	utils.FatalErrorHandle(&err, "error preparing reports sql query")

	reportsOut, err = db.Prepare("SELECT pid, reason, timestamp FROM reports ORDER BY timestamp DESC LIMIT ?, ?")
	utils.FatalErrorHandle(&err, "error preparing reports sql query")

	//BANNED
	bannedTimesOut, err = db.Prepare("SELECT COUNT(*) FROM banned WHERE email_hash=? AND expire_time>?")
	utils.FatalErrorHandle(&err, "error preparing banned sql query")

	banIns, err = db.Prepare("INSERT INTO banned (email_hash, reason, timestamp, expire_time) VALUES (?, ?, ?, ?)")
	utils.FatalErrorHandle(&err, "error preparing banned sql query")

	getBannedOut, err = db.Prepare("SELECT reason, timestamp, expire_time FROM banned WHERE email_hash=? ORDER BY timestamp DESC")
	utils.FatalErrorHandle(&err, "error preparing banned sql query")

	bansOut, err = db.Prepare("SELECT reason, timestamp FROM banned ORDER BY timestamp DESC LIMIT ?, ?")
	utils.FatalErrorHandle(&err, "error preparing banned sql query")

	//ATTENTIONS
	getAttentionPidsOut, err = db.Prepare("SELECT pid FROM attentions WHERE email_hash=? LIMIT 1000")
	utils.FatalErrorHandle(&err, "error preparing attentions sql query")

	AddAttentionIns, err = db.Prepare("INSERT INTO attentions (email_hash,  pid) VALUES (?, ?)")
	utils.FatalErrorHandle(&err, "error preparing attentions sql query")

	RemoveAttentionIns, err = db.Prepare("DELETE FROM attentions WHERE email_hash=? AND pid=?")
	utils.FatalErrorHandle(&err, "error preparing attentions sql query")

	isAttentionOut, err = db.Prepare("SELECT COUNT(*) FROM attentions WHERE email_hash=? AND pid=?")
	utils.FatalErrorHandle(&err, "error preparing attentions sql query")
}

func GetAttentionPids(emailHash string) ([]int, error) {
	var rtn []int
	{
	}
	rows, err := getAttentionPidsOut.Query(emailHash)
	if err != nil {
		return nil, err
	}

	var pid int
	for rows.Next() {
		err := rows.Scan(&pid)
		if err != nil {
			log.Fatal(err)
		}
		rtn = append(rtn, pid)
	}
	err = rows.Err()
	if err != nil {
		return nil, err
	}
	return rtn, nil
}

func IsAttention(dzEmailHash string, pid int) (int, error) {
	rtn := 0
	err := isAttentionOut.QueryRow(dzEmailHash, pid).Scan(&rtn)
	return rtn, err
}

func GetOnePost(pid int) (string, string, int, string, string, string, int, int, int, error) {
	var emailHash, text, tag, typ, filePath string
	var timestamp, likenum, replynum, reportnum int
	err := getOnePostOut.QueryRow(pid).Scan(&emailHash, &text, &timestamp, &tag, &typ, &filePath, &likenum, &replynum, &reportnum)
	if reportnum >= 3 && reportnum < 10 && tag == "" {
		tag = "用户举报较多"
	}
	return emailHash, text, timestamp, tag, typ, filePath, likenum, replynum, reportnum, err
}

func BannedTimesPost(dzEmailHash string, fromTimestamp int) (int, error) {
	bannedTimes := 0
	err := bannedTimesOut.QueryRow(dzEmailHash, fromTimestamp).Scan(&bannedTimes)
	return bannedTimes, err
}

func SaveBanUser(dzEmailHash string, reason string, interval int) error {
	timestamp := int(utils.GetTimeStamp())
	_, err := banIns.Exec(dzEmailHash, reason, timestamp, timestamp+interval)

	return err
}

func parsePostsRows(rows *sql.Rows, err error) ([]interface{}, error) {
	var rtn []interface{}
	if err != nil {
		return nil, err
	}

	var emailHash, text, tag, typ, filePath string
	var timestamp, pid, likenum, replynum, reportnum int
	for rows.Next() {
		err := rows.Scan(&pid, &emailHash, &text, &timestamp, &tag, &typ, &filePath, &likenum, &replynum, &reportnum)
		if err != nil {
			log.Fatal(err)
		}
		if reportnum >= 3 && reportnum < 10 && tag == "" {
			tag = "用户举报较多"
		}
		rtn = append(rtn, gin.H{
			"pid":       pid,
			"text":      text,
			"type":      typ,
			"timestamp": timestamp,
			"reply":     replynum,
			"likenum":   likenum,
			"url":       filePath,
			"tag":       utils.IfThenElse(len(tag) != 0, tag, nil),
		})
	}
	err = rows.Err()
	if err != nil {
		return nil, err
	}
	return rtn, nil
}

func GetNewRegisterCountIn24h() int {
	var rtn int
	_ = db.QueryRow("SELECT COUNT(*) FROM user_info WHERE timestamp>(SELECT MAX(timestamp)-86400 FROM user_info)").Scan(&rtn)
	return rtn
}

func GetUserCount() int {
	var rtn int
	_ = db.QueryRow("SELECT COUNT(*) FROM user_info").Scan(&rtn)
	return rtn
}

func GetPostsByPidList(pids []int) ([]interface{}, error) {
	rows, err := db.Query("SELECT pid, email_hash, text, timestamp, tag, type, file_path, likenum, replynum, reportnum FROM posts WHERE pid IN (" + utils.SplitToString(pids, ",") + ") AND reportnum<10 ORDER BY pid DESC")
	return parsePostsRows(rows, err)
}

func GetHotPosts() ([]interface{}, error) {
	rows, err := hotPostsOut.Query()
	return parsePostsRows(rows, err)
}

func SearchSavedPosts(str string, limitMin int, searchPageSize int) ([]interface{}, error) {
	rows, err := searchOut.Query(str, limitMin, searchPageSize)
	return parsePostsRows(rows, err)
}

func GetDeletedPosts(limitMin int, searchPageSize int) ([]interface{}, error) {
	rows, err := deletedOut.Query(limitMin, searchPageSize)
	return parsePostsRows(rows, err)
}

func GetReports(limitMin int, searchPageSize int) ([]interface{}, error) {
	var rtn []interface{}
	rows, err := reportsOut.Query(limitMin, searchPageSize)
	if err != nil {
		return nil, err
	}

	var reason string
	var pid, timestamp int
	for rows.Next() {
		err := rows.Scan(&pid, &reason, &timestamp)
		if err != nil {
			log.Fatal(err)
		}
		rtn = append(rtn, gin.H{
			"pid":       pid,
			"text":      reason,
			"type":      "text",
			"timestamp": timestamp,
			"reply":     0,
			"likenum":   0,
			"url":       "",
			"tag":       nil,
		})
	}
	err = rows.Err()
	if err != nil {
		return nil, err
	}
	return rtn, nil
}

func GetBans(limitMin int, searchPageSize int) ([]interface{}, error) {
	var rtn []interface{}
	rows, err := bansOut.Query(limitMin, searchPageSize)
	if err != nil {
		return nil, err
	}

	var reason string
	var timestamp int
	for rows.Next() {
		err := rows.Scan(&reason, &timestamp)
		if err != nil {
			log.Fatal(err)
		}
		rtn = append(rtn, gin.H{
			"pid":       0,
			"text":      reason,
			"type":      "text",
			"timestamp": timestamp,
			"reply":     0,
			"likenum":   0,
			"url":       "",
			"tag":       nil,
		})
	}
	err = rows.Err()
	if err != nil {
		return nil, err
	}
	return rtn, nil
}

func GetSavedPosts(pidMin int, pidMax int) ([]interface{}, error) {
	var rtn []interface{}
	rows, err := getPostsOut.Query(pidMin, pidMax)
	if err != nil {
		return nil, err
	}

	var emailHash, text, tag, typ, filePath string
	var timestamp, pid, likenum, replynum, reportnum int
	pinnedPids := utils.GetPinnedPids()
	for rows.Next() {
		err := rows.Scan(&pid, &emailHash, &text, &timestamp, &tag, &typ, &filePath, &likenum, &replynum, &reportnum)
		if err != nil {
			log.Fatal(err)
		}
		if _, ok := utils.ContainsInt(pinnedPids, pid); !ok {
			if reportnum >= 3 && reportnum < 10 && tag == "" {
				tag = "用户举报较多"
			}
			rtn = append(rtn, gin.H{
				"pid":       pid,
				"text":      text,
				"type":      typ,
				"timestamp": timestamp,
				"reply":     replynum,
				"likenum":   likenum,
				"url":       filePath,
				"tag":       utils.IfThenElse(len(tag) != 0, tag, nil),
			})
		}
	}
	err = rows.Err()
	if err != nil {
		return nil, err
	}
	return rtn, nil
}

func GetBannedMsgs(emailHash string) ([]interface{}, error) {
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

func GetSavedComments(pid int) ([]interface{}, error) {
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
			"tag":       utils.IfThenElse(len(tag) != 0, tag, nil),
			"name":      name,
		})
	}
	err = rows.Err()
	if err != nil {
		return nil, err
	}
	return rtn, nil
}

func SaveCode(user string, code string) error {
	timestamp := int32(utils.GetTimeStamp())
	_, err := saveCodeIns.Exec(utils.HashEmail(user), timestamp, code, timestamp, code)

	return err
}

func GetCode(hashedUser string) (string, int64, error) {
	var timestamp int64
	var correctCode string
	err := checkCodeOut.QueryRow(hashedUser).Scan(&timestamp, &correctCode)
	if err != nil {
		return "", -1, err
	}
	return correctCode, timestamp, nil
}

func SaveToken(token string, hashedUser string) error {
	timestamp := int32(utils.GetTimeStamp())
	_, err := saveTokenIns.Exec(hashedUser, token, timestamp, timestamp, token)
	return err
}

func GetCommentNameByEmailHash(emailHash string, pid int) (string, error) {
	var name string
	err := checkCommentNameOut.QueryRow(pid, emailHash).Scan(&name)
	return name, err
}

func GetMaxPid() (int, error) {
	var pid int64
	err := db.QueryRow("SELECT MAX(pid) FROM posts").Scan(&pid)
	return int(pid), err
}

func GetCommentCount(pid int, dzEmailHash string) (int, error) {
	var rtn int64
	err := getCommentCountOut.QueryRow(pid, dzEmailHash).Scan(&rtn)
	return int(rtn), err
}

func SavePost(emailHash string, text string, tag string, typ string, filePath string) (int, error) {
	timestamp := int32(utils.GetTimeStamp())
	res, err := doPostIns.Exec(emailHash, text, timestamp, tag, typ, filePath)
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

func SaveComment(emailHash string, text string, tag string, pid int, name string) (int, error) {
	timestamp := int32(utils.GetTimeStamp())
	res, err := doCommentIns.Exec(emailHash, pid, text, tag, timestamp, name)
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

func SaveReport(emailHash string, reason string, pid int) (int, error) {
	timestamp := int32(utils.GetTimeStamp())
	res, err := doReportIns.Exec(emailHash, pid, reason, timestamp)
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

func GetInfoByToken(token string) (string, error) {
	var emailHash string
	err := getInfoOut.QueryRow(token).Scan(&emailHash)
	return emailHash, err
}
