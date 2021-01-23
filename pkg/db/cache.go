package db

import (
	"github.com/patrickmn/go-cache"
	"strconv"
	"thuhole-go-backend/pkg/structs"
	"time"
)

var commentCache *cache.Cache
var tokenCache *cache.Cache

func initCache() {
	commentCache = cache.New(5*time.Hour, 10*time.Hour)
	tokenCache = cache.New(1*time.Minute, 2*time.Minute)
}

func GetUserWithCache(token string) (structs.User, error) {
	userI, hit := tokenCache.Get(token)
	if hit {
		return userI.(structs.User), nil
	} else {
		var user structs.User
		err := db.Where("token = ?", token).First(&user).Error
		if err == nil {
			tokenCache.SetDefault(token, user)
		}
		return user, err
	}
}

func GetCommentsWithCache(post *structs.Post, now time.Time) ([]structs.Comment, error) {
	pid := post.ID
	if !NeedCacheComment(post, now) {
		return GetComments(pid)
	}

	pidStr := strconv.Itoa(int(pid))
	commentsInterface, hit := commentCache.Get(pidStr)
	if hit {
		return commentsInterface.([]structs.Comment), nil
	} else {
		comments, err := GetComments(pid)
		if err == nil {
			commentCache.SetDefault(pidStr, comments)
		}
		return comments, err
	}
}

func DelCommentCache(pid int) {
	commentCache.Delete(strconv.Itoa(pid))
}

func NeedCacheComment(post *structs.Post, now time.Time) bool {
	return now.Before(post.CreatedAt.AddDate(0, 0, 2))
}
