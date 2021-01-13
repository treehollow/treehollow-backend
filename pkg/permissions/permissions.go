package permissions

import (
	"thuhole-go-backend/pkg/structs"
	"thuhole-go-backend/pkg/utils"
)

func GetPermissionsByPost(user *structs.User, post *structs.Post) []string {
	return getPermissions(user, post, false)
}

func getPermissions(user *structs.User, post *structs.Post, isComment bool) []string {
	rtn := []string{"report"}
	if !isComment {
		rtn = append(rtn, "fold")
	}
	timestamp := utils.GetTimeStamp()
	if (user.Role == structs.AdminRole || user.Role == structs.SuperUserRole ||
		((timestamp-post.CreatedAt.Unix() <= 120) && (user.ID == post.UserID))) && (!post.DeletedAt.Valid) {
		rtn = append(rtn, "delete")
	}

	if user.Role == structs.AdminRole || user.Role == structs.SuperUserRole {
		rtn = append(rtn, "set_tag")
		if post.DeletedAt.Valid {
			rtn = append(rtn, "unban")
			rtn = append(rtn, "undelete_unban")
		} else {
			rtn = append(rtn, "delete_ban")
		}
	} else if (timestamp-post.CreatedAt.Unix() <= 172800) && user.Role == structs.DeleterRole && !post.DeletedAt.Valid {
		rtn = append(rtn, "delete_ban")
	} else if (timestamp-post.CreatedAt.Unix() <= 172800) && user.Role == structs.UnDeleterRole && post.DeletedAt.Valid {
		rtn = append(rtn, "undelete_unban")
	}

	return rtn
}

func GetPermissionsByComment(user *structs.User, comment *structs.Comment) []string {
	return getPermissions(user, &structs.Post{
		DeletedAt: comment.DeletedAt,
		CreatedAt: comment.CreatedAt,
		UserID:    comment.UserID,
	}, true)
}

func GetReportWeight(user *structs.User) int32 {
	return 10
}

func NeedLimiter(user *structs.User) bool {
	return user.Role == structs.NormalUserRole || user.Role == structs.DeleterRole || user.Role == structs.UnDeleterRole
}

func CanViewDeletedPost(user *structs.User) bool {
	return user.Role == structs.AdminRole || user.Role == structs.UnDeleterRole ||
		user.Role == structs.SuperUserRole
}

func GetDeletePostRateLimitIn24h(userRole structs.UserRole) int64 {
	switch userRole {
	case structs.SuperUserRole:
		return 10000
	case structs.AdminRole:
		return 20
	case structs.DeleterRole:
		return 20
	default:
		return 0
	}
}

func CanOverrideBan(user *structs.User) bool {
	return user.Role == structs.AdminRole || user.Role == structs.DeleterRole || user.Role == structs.UnDeleterRole ||
		user.Role == structs.SuperUserRole
}

func CanViewStatistics(user *structs.User) bool {
	return user.Role == structs.SuperUserRole || user.Role == structs.AdminRole
}

func CanViewAllSystemMessages(user *structs.User) bool {
	return user.Role == structs.SuperUserRole || user.Role == structs.AdminRole
}

func CanViewReports(user *structs.User) bool {
	return user.Role == structs.AdminRole || user.Role == structs.DeleterRole || user.Role == structs.UnDeleterRole ||
		user.Role == structs.SuperUserRole
}

func CanViewLogs(user *structs.User) bool {
	return user.Role == structs.SuperUserRole
}

func CanShowHelp(user *structs.User) bool {
	return user.Role == structs.AdminRole || user.Role == structs.DeleterRole || user.Role == structs.UnDeleterRole ||
		user.Role == structs.SuperUserRole
}

func CanShutdown(user *structs.User) bool {
	return user.Role == structs.SuperUserRole
}
