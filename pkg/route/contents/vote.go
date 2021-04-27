package contents

import (
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/iancoleman/orderedmap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"net/http"
	"strconv"
	"treehollow-v3-backend/pkg/base"
	"treehollow-v3-backend/pkg/consts"
	"treehollow-v3-backend/pkg/logger"
	"treehollow-v3-backend/pkg/utils"
)

func sendVote(c *gin.Context) {
	user := c.MustGet("user").(base.User)
	canViewDelete := base.CanViewDeletedPost(&user)
	option := c.PostForm("option")

	pid, err := strconv.Atoi(c.PostForm("pid"))
	if err != nil {
		base.HttpReturnWithCodeMinusOne(c, logger.NewError(err, "SendVoteInvalidPid", "投票操作失败，pid不合法"))
		return
	}

	_ = base.GetDb(false).Transaction(func(tx *gorm.DB) error {
		var post base.Post

		err3 := utils.UnscopedTx(tx, canViewDelete).Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&post, int32(pid)).Error
		if err3 != nil {
			if errors.Is(err3, gorm.ErrRecordNotFound) {
				base.HttpReturnWithCodeMinusOne(c, logger.NewSimpleError("SendVoteNoPid", "投票失败，pid不存在", logger.WARN))
			} else {
				base.HttpReturnWithCodeMinusOne(c, logger.NewError(err3, "SendVoteFailedGetPost", consts.DatabaseReadFailedString))
			}
			return err3
		}

		voteData := orderedmap.New()
		err = json.Unmarshal([]byte(post.VoteData), &voteData)
		if err != nil {
			base.HttpReturnWithCodeMinusOne(c, logger.NewError(err, "BadVoteData", consts.DatabaseDamagedString))
			return nil
		}
		voteOptionCount, optionExist := voteData.Get(option)
		if !optionExist {
			base.HttpReturnWithCodeMinusOne(c, logger.NewSimpleError("VoteNoOption", "投票失败，选项不存在", logger.ERROR))
			return nil
		}

		var count int64
		err3 = tx.Model(&base.Vote{}).Where("user_id = ? and post_id = ?", user.ID, post.ID).
			Count(&count).Error
		if err3 != nil {
			base.HttpReturnWithCodeMinusOne(c, logger.NewError(err3, "SendVoteFailedGetCount", consts.DatabaseReadFailedString))
			return err3
		}
		if count > 0 {
			base.HttpReturnWithCodeMinusOne(c, logger.NewSimpleError("AlreadyVoted", "投票失败，已经投过票了", logger.WARN))
			return errors.New("投票失败，已经投过票了")
		}

		err3 = tx.Create(&base.Vote{
			PostID: post.ID,
			UserID: user.ID,
			Option: option,
		}).Error
		if err3 != nil {
			base.HttpReturnWithCodeMinusOne(c, logger.NewError(err3, "SaveVoteFailed", consts.DatabaseWriteFailedString))
			return err3
		}

		voteData.Set(option, voteOptionCount.(float64)+1)
		_newVoteData, _ := json.Marshal(voteData)
		newVoteData := string(_newVoteData)

		err3 = tx.Table("posts").Where("id = ?", post.ID).
			UpdateColumn("vote_data", newVoteData).Error
		if err3 != nil {
			base.HttpReturnWithCodeMinusOne(c, logger.NewError(err3, "SaveVoteFPostFailed", consts.DatabaseWriteFailedString))
			return err3
		}

		c.JSON(http.StatusOK, gin.H{
			"code": 0,
			"vote": gin.H{
				"voted":        option,
				"vote_options": voteData.Keys(),
				"vote_data":    voteData,
			},
		})

		return nil
	})
}
