package main

import (
	"fmt"
	"gopkg.in/telegram-bot-api.v4"
	"log"
	"os"
	"time"
)

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

	bot.Debug = true

	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := bot.GetUpdatesChan(u)

	ticker := time.NewTicker(time.Second * 30)
	go func() {
		for t := range ticker.C {
			toTea := durationToNextTea()
			log.Printf("to next tea: %d", toTea)

			msg := tgbotapi.NewMessage(0, fmt.Sprintf("До чая осталось: %s.", timeToNextTea().Format("15:04:05")))
			bot.Send(msg)

			fmt.Println("Tick at", t)
		}
	}()

	for update := range updates {

		if update.Message == nil {
			continue
		}

		log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)

		if update.Message.Text == "/чай" || update.Message.Text == "/tea" {
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("До чая осталось: %s.", timeToNextTea().Format("15:04:05")))
			msg.ReplyToMessageID = update.Message.MessageID
			bot.Send(msg)
		} else {
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "скорее всего конечно же нет")
			msg.ReplyToMessageID = update.Message.MessageID
			bot.Send(msg)
		}
	}
}
