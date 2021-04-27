package push

import (
	"encoding/json"
	"github.com/sideshow/apns2"
	"github.com/sideshow/apns2/certificate"
	"github.com/sideshow/apns2/payload"
	"github.com/sideshow/apns2/token"
	"github.com/spf13/viper"
	"log"
	"strconv"
	"strings"
	"treehollow-v3-backend/pkg/base"
	"treehollow-v3-backend/pkg/model"
	"treehollow-v3-backend/pkg/utils"
)

func getIOSPushClient() (*apns2.Client, error) {
	authFile := viper.GetString("ios_push_auth_file")
	if strings.HasSuffix(authFile, "p12") {
		certPass := viper.GetString("ios_push_auth_password")
		cert, err := certificate.FromP12File(authFile, certPass)
		if err != nil {
			log.Printf("ios push cert error: %s\n", err)
			return nil, err
		}
		return apns2.NewClient(cert), nil
	}
	authKey, err := token.AuthKeyFromFile(authFile)
	if err != nil {
		log.Printf("ios push token error: %s\n", err)
		return nil, err
	}
	t := &token.Token{
		AuthKey: authKey,
		// KeyID from developer account (Certificates, Identifiers & Profiles -> Keys)
		KeyID: viper.GetString("ios_push_key_id"),
		// TeamID from developer account (View Account -> Membership)
		TeamID: viper.GetString("ios_push_team_id"),
	}
	pushClient := apns2.NewTokenClient(t)
	return pushClient, nil
}

func SendMessages(msgs []base.PushMessage, Api *API, isDoRecall bool) {
	var err error

	if !isDoRecall {
		err = base.GetDb(false).Create(&msgs).Error
		if err != nil {
			log.Printf("create push messages failed: %s", err)
			return
		}
	}

	pushUserIDs := make([]int32, 0, len(msgs))
	pushMap := make(map[int32]*base.PushMessage)
	for _, msg := range msgs {
		if msg.DoPush {
			pushUserIDs = append(pushUserIDs, msg.UserID)
			pushMap[msg.UserID] = &msg
		}
	}

	var devices []base.Device
	err = base.GetDb(false).Model(&base.Device{}).Where("user_id in (?)", pushUserIDs).
		Find(&devices).Error
	if err != nil {
		log.Printf("read push devices failed: %s", err)
		return
	}

	pushClient, err := getIOSPushClient()
	if err != nil {
		log.Printf("getIOSPushClient error: %s\n", err)
		return
	}

	//TODO: (middle priority)fix "recall before push" bug
	for _, device := range devices {
		msg := pushMap[device.UserID]
		switch device.Type {
		case base.IOSDevice:
			if len(device.IOSDeviceToken) > 0 {
				var p *payload.Payload
				if isDoRecall {
					p = payload.NewPayload().AlertTitle("消息已被删除").Custom("delete", 1).
						Custom("pid", msg.PostID)
				} else {
					p = payload.NewPayload().AlertTitle(utils.TrimText(msg.Title, 50)).
						AlertBody(utils.TrimText(msg.Message, 100)).Sound("default")
					if (msg.Type & (model.ReplyMeComment | model.CommentInFavorited)) > 0 {
						p = p.Custom("pid", msg.PostID).Custom("cid", msg.CommentID)
					}
					p.Custom("type", msg.Type)
					if viper.GetBool("is_debug") {
						log.Printf("ios push notification: %v\n", msg)
					}
				}
				res, err2 := pushClient.Production().Push(&apns2.Notification{
					DeviceToken: device.IOSDeviceToken,
					Topic:       "treehollow.Hollow",
					Payload:     p,
					CollapseID:  strconv.Itoa(int(msg.ID)),
				})

				if err2 != nil {
					log.Printf("production push ios notifation failed: %s", err2)
				}

				if viper.GetBool("is_debug") {
					if res != nil {
						log.Printf("production push notification response: %s %d, %s\n", p, res.StatusCode, res.Reason)
					}

					res2, err2 := pushClient.Development().Push(&apns2.Notification{
						DeviceToken: device.IOSDeviceToken,
						Topic:       "treehollow.Hollow",
						Payload:     p,
						CollapseID:  strconv.Itoa(int(msg.ID)),
					})

					if err2 != nil {
						log.Printf("dev push ios notifation failed: %s", err2)
					}

					if res2 != nil {
						log.Printf("dev push notification response: %s %d, %s\n", p, res2.StatusCode, res2.Reason)
					}
				}
			}
		case base.AndroidDevice:
			var p map[string]interface{}
			if isDoRecall {
				p = map[string]interface{}{
					"id":     msg.ID,
					"delete": 1,
				}
			} else {
				p = map[string]interface{}{
					"id":        msg.ID,
					"title":     msg.Title,
					"body":      utils.TrimText(msg.Message, 100),
					"type":      msg.Type,
					"timestamp": msg.UpdatedAt.Unix(),
				}
				if (msg.Type & (model.ReplyMeComment | model.CommentInFavorited)) > 0 {
					p["pid"] = msg.PostID
					p["cid"] = msg.CommentID
				}
			}
			postBody, _ := json.Marshal(p)
			Api.Notify(device.Token, &postBody)
		}
	}
}
