package model

import "time"

type Message struct {
	Message string
	Title   string
	Extras  map[string]interface{}
	Time    time.Time
}

type PushType int8

const (
	SystemMessage      PushType = 0x01
	ReplyMeComment     PushType = 0x02
	CommentInFavorited PushType = 0x04
)

type SearchOrder int8

const (
	SearchOrderByID       SearchOrder = 0
	SearchOrderByLikeNum  SearchOrder = 1
	SearchOrderByReplyNum SearchOrder = 2
)

func SearchOrderFromString(s string) (searchOrder SearchOrder) {
	switch s {
	case "id":
		searchOrder = SearchOrderByID
	case "like_num":
		searchOrder = SearchOrderByLikeNum
	case "reply_num":
		searchOrder = SearchOrderByReplyNum
	default:
		searchOrder = SearchOrderByID
	}
	return
}

func (searchOrder *SearchOrder) ToString() string {
	switch *searchOrder {
	case SearchOrderByLikeNum:
		return "like_num desc"
	case SearchOrderByReplyNum:
		return "reply_num desc"
	case SearchOrderByID:
		return "id desc"
	default:
		return "id desc"
	}
}
