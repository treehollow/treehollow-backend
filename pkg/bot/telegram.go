package bot

import (
	"github.com/spf13/viper"
	tb "gopkg.in/tucnak/telebot.v2"
	"time"
	"treehollow-v3-backend/pkg/utils"
)

type TgMessage struct {
	Text      string
	ImagePath string
}

var TgMessageChannel = make(chan TgMessage)

func InitBot() {
	if viper.GetBool("enable_telegram") {
		poller := &tb.LongPoller{Timeout: 10 * time.Second}
		filteredPoller := tb.NewMiddlewarePoller(poller, func(upd *tb.Update) bool {
			if upd.Message == nil {
				return false
			}

			if upd.Message.Chat.ID == viper.GetInt64("tg_chat_id") {
				return true
			}

			return false
		})

		b, err := tb.NewBot(tb.Settings{
			// You can also set custom API URL.
			// If field is empty it equals to "https://api.telegram.org".
			//URL: "http://195.129.111.17:8012",

			Token:  viper.GetString("tg_token"),
			Poller: filteredPoller,
		})

		if err != nil {
			utils.FatalErrorHandle(&err, "Telegram bot init failed")
			return
		}

		b.Handle("/ping", func(m *tb.Message) {
			_, _ = b.Send(m.Sender, "pong!")
		})

		go func() {
			_, _ = b.Send(tb.ChatID(viper.GetInt64("tg_chat_id")), "Backend bot started!")
			b.Start()
		}()

		go func() {
			for m := range TgMessageChannel {
				if len(m.ImagePath) == 0 {
					_, _ = b.Send(tb.ChatID(viper.GetInt64("tg_chat_id")), utils.TrimText(m.Text, 4096))
				} else {
					_, _ = b.Send(tb.ChatID(viper.GetInt64("tg_chat_id")),
						&tb.Photo{File: tb.FromDisk(m.ImagePath), Caption: utils.TrimText(m.Text, 1024)})
				}
			}
		}()
	}
}
