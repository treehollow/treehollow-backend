package base

import (
	"treehollow-v3-backend/pkg/utils"
)

func GetPermissionsByPost(user *User, post *Post) []string {
	return getPermissions(user, post, false)
}

func isDeleter(role UserRole) bool {
	return role == DeleterRole || role == Deleter2Role || role == Deleter3Role
}

func getPermissions(user *User, post *Post, isComment bool) []string {
	rtn := []string{"report"}
	if !isComment {
		rtn = append(rtn, "fold")
	}
	timestamp := utils.GetTimeStamp()
	if (user.Role == AdminRole || user.Role == SuperUserRole ||
		((timestamp-post.CreatedAt.Unix() <= 120) && (user.ID == post.UserID))) && (!post.DeletedAt.Valid) {
		rtn = append(rtn, "delete")
	}

	if user.Role == AdminRole || user.Role == SuperUserRole {
		rtn = append(rtn, "set_tag")
		if post.DeletedAt.Valid {
			rtn = append(rtn, "unban")
			rtn = append(rtn, "undelete_unban")
		} else {
			rtn = append(rtn, "delete_ban")
		}
	} else if (timestamp-post.CreatedAt.Unix() <= 172800) && isDeleter(user.Role) && !post.DeletedAt.Valid {
		rtn = append(rtn, "delete_ban")
	} else if (timestamp-post.CreatedAt.Unix() <= 172800) && user.Role == UnDeleterRole && post.DeletedAt.Valid {
		rtn = append(rtn, "undelete_unban")
	}

	return rtn
}

func GetPermissionsByComment(user *User, comment *Comment) []string {
	return getPermissions(user, &Post{
		DeletedAt: comment.DeletedAt,
		CreatedAt: comment.CreatedAt,
		UserID:    comment.UserID,
	}, true)
}

func GetReportWeight(user *User) int32 {
	return 10
}

func NeedLimiter(user *User) bool {
	return user.Role == NormalUserRole || isDeleter(user.Role) || user.Role == UnDeleterRole
}

func CanViewDeletedPost(user *User) bool {
	return user.Role == AdminRole || user.Role == UnDeleterRole ||
		user.Role == SuperUserRole
}

func GetDeletePostRateLimitIn24h(userRole UserRole) int64 {
	switch userRole {
	case SuperUserRole:
		return 10000
	case AdminRole:
		return 20
	case DeleterRole:
		return 20
	case Deleter2Role:
		return 5
	case Deleter3Role:
		return 0
	default:
		return 0
	}
}

func CanOverrideBan(user *User) bool {
	return user.Role == AdminRole || isDeleter(user.Role) || user.Role == UnDeleterRole ||
		user.Role == SuperUserRole
}

func CanViewStatistics(user *User) bool {
	return user.Role == SuperUserRole || user.Role == AdminRole
}

func CanViewAllSystemMessages(user *User) bool {
	return user.Role == SuperUserRole || user.Role == AdminRole
}

func CanViewReports(user *User) bool {
	return user.Role == AdminRole || isDeleter(user.Role) || user.Role == UnDeleterRole ||
		user.Role == SuperUserRole
}

func CanViewLogs(user *User) bool {
	return user.Role == SuperUserRole
}

func CanShowHelp(user *User) bool {
	return user.Role == AdminRole || isDeleter(user.Role) || user.Role == UnDeleterRole ||
		user.Role == SuperUserRole
}

func CanShutdown(user *User) bool {
	return user.Role == SuperUserRole
}

func CanViewDecryptionMessages(user *User) bool {
	return user.Role == SuperUserRole
}
