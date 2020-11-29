package permissions

import (
	"thuhole-go-backend/pkg/structs"
	"thuhole-go-backend/pkg/utils"
)

func GetPermissionsByPost(user structs.User, post structs.Post) []string {
	rtn := []string{"fold", "report"}
	timestamp := utils.GetTimeStamp()
	if (user.Role == structs.AdminRole || user.Role == structs.DeleterRole || user.Role == structs.SuperUserRole ||
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
	} else if user.Role == structs.DeleterRole && !post.DeletedAt.Valid {
		rtn = append(rtn, "delete_ban")
	} else if user.Role == structs.UnDeleterRole && post.DeletedAt.Valid {
		rtn = append(rtn, "unban")
		rtn = append(rtn, "undelete_unban")
	}

	return rtn
}

func GetPermissionsByComment(user structs.User, comment structs.Comment) []string {
	return GetPermissionsByPost(user, structs.Post{
		DeletedAt: comment.DeletedAt,
		CreatedAt: comment.CreatedAt,
		UserID:    comment.UserID,
	})
}

func GetReportWeight(user structs.User) int32 {
	return 10
}

func NeedLimiter(user structs.User) bool {
	return user.Role == structs.NormalUserRole || user.Role == structs.DeleterRole || user.Role == structs.UnDeleterRole
}

func CanViewDeletedPost(user structs.User) bool {
	return user.Role == structs.AdminRole || user.Role == structs.UnDeleterRole ||
		user.Role == structs.SuperUserRole
}

func CanOverrideBan(user structs.User) bool {
	return user.Role == structs.SuperUserRole || user.Role == structs.AdminRole
}

func CanViewStatistics(user structs.User) bool {
	return user.Role == structs.SuperUserRole || user.Role == structs.AdminRole
}

func CanViewAllSystemMessages(user structs.User) bool {
	return user.Role == structs.SuperUserRole || user.Role == structs.AdminRole
}

func CanViewReports(user structs.User) bool {
	return user.Role == structs.AdminRole || user.Role == structs.DeleterRole || user.Role == structs.UnDeleterRole ||
		user.Role == structs.SuperUserRole
}

func CanShowHelp(user structs.User) bool {
	return user.Role == structs.AdminRole || user.Role == structs.DeleterRole || user.Role == structs.UnDeleterRole ||
		user.Role == structs.SuperUserRole
}

func CanShutdown(user structs.User) bool {
	return user.Role == structs.SuperUserRole
}
