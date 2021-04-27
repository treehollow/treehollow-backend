package base

import (
	"context"
	"github.com/go-redis/cache/v8"
	"gorm.io/gorm"
	"log"
	"strconv"
	"time"
	"treehollow-v3-backend/pkg/consts"
	"treehollow-v3-backend/pkg/logger"
	"treehollow-v3-backend/pkg/utils"
)

var tokenCache *cache.Cache
var commentCache *cache.Cache

const CommentCacheExpireTime = 5 * time.Hour
const TOKENCacheExpireTime = 1 * time.Minute

func initCache() {
	tokenCache = cache.New(&cache.Options{Redis: redisClient})
	commentCache = cache.New(&cache.Options{Redis: redisClient})
}

func GetUserWithCache(token string) (User, error) {
	ctx := context.TODO()
	var user User
	err := tokenCache.Get(ctx, "token"+token, &user)
	if err == nil {
		return user, nil
	} else {
		subQuery := db.Model(&Device{}).Distinct().
			Where("token = ? and created_at > ?", token, utils.GetEarliestAuthenticationTime()).
			Select("user_id")
		err = db.Where("id = (?)", subQuery).First(&user).Error
		if err == nil {
			err = tokenCache.Set(&cache.Item{
				Ctx:   ctx,
				Key:   "token" + token,
				Value: &user,
				TTL:   TOKENCacheExpireTime,
			})
		}
		return user, err
	}
}

func DelUserCache(token string) error {
	ctx := context.TODO()
	err := tokenCache.Delete(ctx, "token"+token)
	if err != nil {
		log.Printf("DelUserCache error: %s\n", err)
	}
	return err
}

func GetCommentsWithCache(post *Post, now time.Time) ([]Comment, error) {
	pid := post.ID
	if !NeedCacheComment(post, now) {
		return GetComments(pid)
	}

	ctx := context.TODO()
	pidStr := strconv.Itoa(int(pid))
	var comments []Comment
	err := commentCache.Get(ctx, "pid"+pidStr, &comments)
	if err == nil {
		return comments, err
	} else {
		comments, err = GetComments(pid)
		if err == nil {
			err = commentCache.Set(&cache.Item{
				Ctx:   ctx,
				Key:   "pid" + pidStr,
				Value: &comments,
				TTL:   CommentCacheExpireTime,
			})
		}
		return comments, err
	}
}

func GetMultipleCommentsWithCache(tx *gorm.DB, posts []Post, now time.Time) (map[int32][]Comment, *logger.InternalError) {
	ctx := context.TODO()
	rtn := make(map[int32][]Comment)
	noCachePids := make(map[int32]bool)
	var noCachePidsArray []int32
	for _, post := range posts {
		pid := post.ID
		if !NeedCacheComment(&post, now) {
			noCachePids[pid] = true
			noCachePidsArray = append(noCachePidsArray, pid)
			continue
		}

		pidStr := strconv.Itoa(int(pid))
		var comments []Comment
		err := commentCache.Get(ctx, "pid"+pidStr, &comments)
		if err == nil {
			rtn[pid] = comments
		} else {
			noCachePids[pid] = true
			noCachePidsArray = append(noCachePidsArray, pid)
			continue
		}
	}

	if len(noCachePidsArray) > 0 {
		comments, err := GetMultipleComments(tx, noCachePidsArray)
		if err != nil {
			return nil, logger.NewError(err, "SQLGetMultipleCommentsFailed", consts.DatabaseReadFailedString)
		}
		for _, comment := range comments {
			rtn[comment.PostID] = append(rtn[comment.PostID], comment)
		}
	}
	for _, post := range posts {
		pid := post.ID
		if _, noCache := noCachePids[pid]; noCache {
			comments2, commentsExist := rtn[pid]
			if !commentsExist {
				comments2 = []Comment{}
			}
			err := commentCache.Set(&cache.Item{
				Ctx:   ctx,
				Key:   "pid" + strconv.Itoa(int(pid)),
				Value: &comments2,
				TTL:   CommentCacheExpireTime,
			})
			if err != nil {
				return nil, logger.NewError(err, "CommentCacheSetFailed", consts.DatabaseReadFailedString)
			}
		}
	}
	return rtn, nil
}

func DelCommentCache(pid int) error {
	ctx := context.TODO()
	err := commentCache.Delete(ctx, "pid"+strconv.Itoa(pid))
	if err != nil {
		log.Printf("DelCommentCache error: %s\n", err)
	}
	return err
}

func NeedCacheComment(post *Post, now time.Time) bool {
	return now.Before(post.CreatedAt.AddDate(0, 0, 365))
	//return now.Before(post.CreatedAt.AddDate(0, 0, 2))
}
