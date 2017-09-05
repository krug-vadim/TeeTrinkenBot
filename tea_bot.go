package main

import (
	"fmt"
	"gopkg.in/telegram-bot-api.v4"
	"log"
	"os"
	"time"
	"strconv"
)

const TeaTimeDuration = time.Minute*20

func durationToNextTea() time.Duration {
	t := time.Now()
	hour := t.Hour()
	next_hour := ((hour+1)/2)*2 + 1
	if next_hour >= 24 {
		next_hour -= 24
	}
	next_tea := time.Date(t.Year(), t.Month(), t.Day(), next_hour, 0, 0, 0, time.Local)
	return next_tea.Sub(t)
}

func timeToNextTea() time.Time {
	return time.Time{}.Add(durationToNextTea())
}

func main() {
	bot, err := tgbotapi.NewBotAPI(os.Getenv("TEABOT_TOKEN"))
	if err != nil {
		log.Panic(err)
	}

	teaTimeChatId, err := strconv.ParseInt(os.Getenv("TEABOT_CHAT"), 10, 64)
	if err != nil {
		log.Panic(err)
	}

	log.Printf("Tea group chat id is %x", teaTimeChatId)

	bot.Debug = false

	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := bot.GetUpdatesChan(u)

	need_create_timer_to_send_alarm := true
	tea_time_ongoing := false

	ticker := time.NewTicker(time.Second * 10)
	go func() {
		for t := range ticker.C {
			toTea := durationToNextTea()
			log.Printf("to next tea: %d", toTea)

			fmt.Println("Tick at", t)

			if ( need_create_timer_to_send_alarm ) {
				need_create_timer_to_send_alarm = false
				timer := time.NewTimer(toTea)
				go func() {
					<- timer.C
					msg := tgbotapi.NewMessage(teaTimeChatId, "го чай")
					bot.Send(msg)
					need_create_timer_to_send_alarm = true

					tea_time_ongoing = true
					tea_ongoin := time.NewTimer(TeaTimeDuration)
					go func() {
						<- tea_ongoin.C
						tea_time_ongoing = false
					}()
				}()
			}
		}
	}()

	for update := range updates {

		if update.Message == nil {
			continue
		}

		log.Printf("[%s/%d] %s", update.Message.From.UserName, update.Message.Chat.ID, update.Message.Text)

		if update.Message.Text == "/чай" || update.Message.Text == "/tea" {
			msg_txt := ""
			if ( tea_time_ongoing ) {
				msg_txt = fmt.Sprintf("Нормальные люди уже пьют чай.")
			} else {
				msg_txt = fmt.Sprintf("До чая осталось: %s.", timeToNextTea().Format("15:04:05"))
			}
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, msg_txt)
			msg.ReplyToMessageID = update.Message.MessageID
			bot.Send(msg)
		} else {
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "скорее всего конечно же нет")
			msg.ReplyToMessageID = update.Message.MessageID
			bot.Send(msg)
		}
	}
}
