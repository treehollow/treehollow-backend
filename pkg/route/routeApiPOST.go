package route

import (
	"bytes"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"thuhole-go-backend/pkg/consts"
	"thuhole-go-backend/pkg/db"
	"thuhole-go-backend/pkg/s3"
	"thuhole-go-backend/pkg/utils"
	"unicode/utf8"
)

func generateTag(text string) string {
	re := regexp.MustCompile(`[#＃](性相关|政治相关|引战|未经证实的传闻|令人不适|NSFW|nsfw|折叠)`)
	if re.MatchString(text) {
		return strings.ToUpper(re.FindStringSubmatch(text)[1])
	}
	re1, err := regexp.Compile(viper.GetString("fold_regex"))
	if err == nil && re1.MatchString(text) {
		return "折叠"
	}
	re2, err2 := regexp.Compile(viper.GetString("sex_related_regex"))
	if err2 == nil && re2.MatchString(text) {
		return "性相关"
	}
	return ""
}

func doPost(c *gin.Context) {
	text := c.PostForm("text")
	typ := c.PostForm("type")
	token := c.PostForm("user_token")
	img := c.PostForm("data")
	if utf8.RuneCountInString(text) > consts.PostMaxLength {
		utils.HttpReturnWithCodeOne(c, "字数过长！字数限制为"+strconv.Itoa(consts.PostMaxLength)+"字。")
		return
	} else if len(text) == 0 && typ == "text" {
		utils.HttpReturnWithCodeOne(c, "请输入内容")
		return
	} else if typ != "text" && typ != "image" {
		utils.HttpReturnWithCodeOne(c, "未知类型的树洞")
		return
	} else if int(float64(len(img))/consts.Base64Rate) > consts.ImgMaxLength {
		utils.HttpReturnWithCodeOne(c, "图片大小超出限制！")
		return
	}

	emailHash, err3 := db.GetInfoByToken(token)
	if err3 != nil {
		utils.HttpReturnWithCodeOne(c, "发送失败，请检查登录状态")
		return
	}

	context, err5 := postLimiter.Get(c, emailHash)
	if err5 != nil {
		c.AbortWithStatus(500)
		return
	}
	if context.Reached {
		//log.Printf("post limiter reached")
		utils.HttpReturnWithCodeOne(c, "请不要短时间内连续发送树洞")
		return
	}

	context, err5 = postLimiter2.Get(c, emailHash)
	if err5 != nil {
		c.AbortWithStatus(500)
		return
	}
	if context.Reached {
		log.Printf("post limiter 2 reached")
		utils.HttpReturnWithCodeOne(c, "你24小时内已经发送太多树洞了")
		return
	}

	timestamp := int(utils.GetTimeStamp())
	bannedTimes, _ := db.BannedTimesPost(emailHash, timestamp)
	if bannedTimes > 0 {
		utils.HttpReturnWithCodeOne(c, "很抱歉，您当前处于禁言状态，无法发送树洞。")
		return
	}

	tag := generateTag(text)

	var pid int
	var err error
	var imgPath string
	var uploadChan chan bool
	uploadChan = nil
	if typ == "image" {
		imgPath = utils.GenToken()
		sDec, suffix, err2 := utils.SaveImage(img, imgPath)
		if err2 != nil {
			utils.HttpReturnWithCodeOne(c, err2.Error())
			return
		}

		pid, err = db.SavePost(emailHash, text, timestamp, tag, typ, imgPath+suffix)
		if err == nil && len(viper.GetString("DCSecretKey")) > 0 {
			uploadChan = make(chan bool, 1)
			go func() {
				err4 := s3.Upload(imgPath[:2]+"/"+imgPath+suffix, bytes.NewReader(sDec))
				if err4 != nil {
					log.Printf("S3 upload failed, err=%s\n", err4)
				}
				uploadChan <- true
			}()
		}
	} else {
		pid, err = db.SavePost(emailHash, text, timestamp, tag, typ, "")
	}

	if err != nil {
		log.Printf("error dbSavePost! %s\n", err)
		utils.HttpReturnWithCodeOne(c, "数据库写入失败，请联系管理员")
		return
	} else {
		if uploadChan != nil {
			//wait until upload complete.
			<-uploadChan
		}

		_, _ = db.AddAttentionIns.Exec(emailHash, pid)
		_, _ = db.PlusOneAttentionIns.Exec(pid)
		c.JSON(http.StatusOK, gin.H{
			"code": 0,
			"data": pid,
		})
		return
	}
}

func doComment(c *gin.Context) {
	text := c.PostForm("text")
	typ := c.PostForm("type")
	token := c.PostForm("user_token")
	img := c.PostForm("data")
	if typ != "image" {
		typ = "text"
	}
	if utf8.RuneCountInString(text) > consts.CommentMaxLength {
		utils.HttpReturnWithCodeOne(c, "字数过长！字数限制为"+strconv.Itoa(consts.CommentMaxLength)+"字。")
		return
	} else if len(text) == 0 && typ == "text" {
		utils.HttpReturnWithCodeOne(c, "请输入内容")
		return
	} else if int(float64(len(img))/consts.Base64Rate) > consts.ImgMaxLength {
		utils.HttpReturnWithCodeOne(c, "图片大小超出限制！")
		return
	}
	pid, err := strconv.Atoi(c.PostForm("pid"))
	if err != nil {
		utils.HttpReturnWithCodeOne(c, "发送失败，pid不合法")
		return
	}
	emailHash, err5 := db.GetInfoByToken(token)
	if err5 != nil {
		utils.HttpReturnWithCodeOne(c, "发送失败，请检查登录状态")
		return
	}

	context, err6 := commentLimiter.Get(c, emailHash)
	if err6 != nil {
		c.AbortWithStatus(500)
		return
	}
	if context.Reached {
		//log.Printf("comment limiter reached")
		utils.HttpReturnWithCodeOne(c, "请不要短时间内连续发送树洞回复")
		return
	}

	context, err6 = commentLimiter2.Get(c, emailHash)
	if err6 != nil {
		c.AbortWithStatus(500)
		return
	}
	if context.Reached {
		log.Printf("commment limiter 2 reached")
		utils.HttpReturnWithCodeOne(c, "你24小时内已经发送太多树洞回复了")
		return
	}

	timestamp := int(utils.GetTimeStamp())
	bannedTimes, _ := db.BannedTimesPost(emailHash, timestamp)
	if bannedTimes > 0 {
		utils.HttpReturnWithCodeOne(c, "很抱歉，您当前处于禁言状态，无法发送评论。")
		return
	}
	dzEmailHash, _, _, _, _, _, _, _, _, err := db.GetOnePost(pid)
	if err != nil {
		utils.HttpReturnWithCodeOne(c, "发送失败，pid不存在")
		return
	}

	var name string
	var names0, names1 []string

	names0, names1 = consts.Names0, consts.Names1
	commentMux.Lock()

	name, err = db.GenCommenterName(dzEmailHash, emailHash, pid, names0, names1)
	if err != nil {
		log.Printf("error GenCommenterName in doComment(), err=%s\n", err.Error())
		utils.HttpReturnWithCodeOne(c, "数据库读取失败，请联系管理员")
		commentMux.Unlock()
		return
	}

	var imgPath string
	var uploadChan chan bool
	uploadChan = nil
	if typ == "image" {
		imgPath = utils.GenToken()
		sDec, suffix, err2 := utils.SaveImage(img, imgPath)
		if err2 != nil {
			utils.HttpReturnWithCodeOne(c, err2.Error())
			commentMux.Unlock()
			return
		}

		_, err = db.SaveComment(emailHash, text, "", timestamp, typ, imgPath+suffix, pid, name)
		if err == nil && len(viper.GetString("DCSecretKey")) > 0 {
			uploadChan = make(chan bool, 1)
			go func() {
				err4 := s3.Upload(imgPath[:2]+"/"+imgPath+suffix, bytes.NewReader(sDec))
				if err4 != nil {
					log.Printf("S3 upload failed, err=%s\n", err4)
				}
				uploadChan <- true
			}()
		}
	} else {
		_, err = db.SaveComment(emailHash, text, "", timestamp, "text", "", pid, name)
	}
	commentMux.Unlock()

	if err != nil {
		utils.HttpReturnWithCodeOne(c, "数据库写入失败，请联系管理员")
		return
	} else {

		_, err = db.PlusOneCommentIns.Exec(pid)
		if err != nil {
			log.Printf("error plusOneCommentIns while commenting: %s\n", err)
		}
		isAttention, err := db.IsAttention(emailHash, pid)
		if err == nil && isAttention == 0 {
			_, _ = db.AddAttentionIns.Exec(emailHash, pid)
			_, _ = db.PlusOneAttentionIns.Exec(pid)
		}

		// set tag
		if dzEmailHash == emailHash {
			re := regexp.MustCompile(`[#＃](性相关|政治相关|引战|未经证实的传闻|令人不适|NSFW|nsfw|折叠|重复内容)`)
			if re.MatchString(text) {
				tag := strings.ToUpper(re.FindStringSubmatch(text)[1])
				_, _ = db.SetPostTagIns.Exec(tag, pid)
			}
		}

		if uploadChan != nil {
			//wait until upload complete.
			<-uploadChan
		}

		c.JSON(http.StatusOK, gin.H{
			"code": 0,
			"data": pid,
		})
	}
}

func doReport(c *gin.Context) {
	reason := c.PostForm("reason")
	if len(reason) > consts.ReportMaxLength {
		utils.HttpReturnWithCodeOne(c, "字数过长！字数限制为"+strconv.Itoa(consts.ReportMaxLength)+"字节。")
		return
	} else if len(reason) == 0 {
		utils.HttpReturnWithCodeOne(c, "请输入内容")
		return
	}
	pid, err := strconv.Atoi(c.PostForm("pid"))
	if err != nil {
		utils.HttpReturnWithCodeOne(c, "举报失败，pid不合法")
		return
	} else if _, ok := utils.ContainsInt(viper.GetIntSlice("disallow_report_pids"), pid); ok {
		utils.HttpReturnWithCodeOne(c, "举报失败，哈哈")
		return
	}
	token := c.PostForm("user_token")
	dzEmailHash, text, _, _, typ, _, _, _, reportnum, err2 := db.GetOnePost(pid)
	if err2 != nil {
		utils.HttpReturnWithCodeOne(c, "举报失败，pid不存在")
		return
	}
	emailHash, err5 := db.GetInfoByToken(token)
	if err5 != nil {
		utils.HttpReturnWithCodeOne(c, "举报失败，请检查登录状态")
		return
	}
	_, err = db.SaveReport(emailHash, reason, pid)
	if err != nil {
		utils.HttpReturnWithCodeOne(c, "举报失败")
		return
	} else {
		if reportnum == 9 {
			_, err = db.PlusReportIns.Exec(1, pid)
			if err != nil {
				log.Printf("error plusOneReportIns while reporting: %s\n", err)
			}

			msg := "您的" + typ + "树洞" + strconv.Itoa(pid) + "\n\"" + text + "\"\n由于用户举报过多被删除。"

			err = db.BanUser(dzEmailHash, msg)
			if err != nil {
				log.Printf("error dbSaveBanUser while reporting: %s\n", err)
			}
		} else if _, isAdmin := utils.ContainsString(viper.GetStringSlice("admins_tokens"), token); isAdmin {
			_, err = db.PlusReportIns.Exec(666, pid)
			if err != nil {
				log.Printf("error plus666ReportIns while reporting: %s\n", err)
			}

			msg := "您的" + typ + "树洞" + strconv.Itoa(pid) + "\n\"" + text + "\"\n被管理员删除。管理员的删除理由是：【" + reason + "】。"

			err = db.BanUser(dzEmailHash, msg)
			if err != nil {
				log.Printf("error dbSaveBanUser while reporting: %s\n", err)
			}
		} else {
			_, err = db.PlusReportIns.Exec(1, pid)
			if err != nil {
				log.Printf("error plusOneReportIns while reporting: %s\n", err)
			}
		}

		if err != nil {
			utils.HttpReturnWithCodeOne(c, "举报失败，请联系管理员")
			return
		} else {
			c.JSON(http.StatusOK, gin.H{
				"code": 0,
			})
		}
	}
}

func doAttention(c *gin.Context) {
	pid, err := strconv.Atoi(c.PostForm("pid"))
	if err != nil {
		utils.HttpReturnWithCodeOne(c, "关注操作失败，pid不合法")
		return
	}
	_, _, _, _, _, _, _, _, _, err3 := db.GetOnePost(pid)
	if err3 != nil {
		utils.HttpReturnWithCodeOne(c, "关注失败，pid不存在")
		return
	}
	s := c.PostForm("switch")
	token := c.PostForm("user_token")
	emailHash, err5 := db.GetInfoByToken(token)
	if err5 != nil {
		utils.HttpReturnWithCodeOne(c, "关注失败，请检查登录状态")
		return
	}

	context, err6 := doAttentionLimiter.Get(c, emailHash)
	if err6 != nil {
		c.AbortWithStatus(500)
		return
	}
	if context.Reached {
		log.Printf("do_attention limiter limiter reached")
		utils.HttpReturnWithCodeOne(c, "你今天关注太多树洞了，明天再来吧")
		return
	}

	isAttention, err2 := db.IsAttention(emailHash, pid)
	if err2 != nil {
		log.Printf("error dbIsAttention while doAttention: %s\n", err2)
		utils.HttpReturnWithCodeOne(c, "数据库读取失败，请联系管理员")
		return
	}
	if isAttention == 0 && s == "0" {
		utils.HttpReturnWithCodeOne(c, "您已经取消关注了")
		return
	}
	if isAttention == 1 && s == "1" {
		utils.HttpReturnWithCodeOne(c, "您已经关注过了")
		return
	}
	if isAttention == 0 {
		_, _ = db.AddAttentionIns.Exec(emailHash, pid)
		_, _ = db.PlusOneAttentionIns.Exec(pid)
	}
	if isAttention == 1 {
		_, _ = db.RemoveAttentionIns.Exec(emailHash, pid)
		_, _ = db.MinusOneAttentionIns.Exec(pid)
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
	})
}

func apiPost(c *gin.Context) {
	action := c.Query("action")
	switch action {
	case "docomment":
		doComment(c)
		return
	case "dopost":
		doPost(c)
		return
	case "attention":
		doAttention(c)
		return
	case "report":
		doReport(c)
		return
	default:
		c.AbortWithStatus(403)
	}
}
