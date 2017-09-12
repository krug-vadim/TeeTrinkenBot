package main

import (
	"gopkg.in/telegram-bot-api.v4"
	"fmt"
	"log"
	"time"
	"strings"
	"io/ioutil"
	"encoding/json"
)

type entry struct {
	Day   []string `json:"day"`
	Time  []string `json:"time"`
}

type bot_config struct {
	BotToken   string           `json:"tea_time_bot_token"`
	ChatId     int64            `json:"tea_time_chat_id"`
	Schedule   map[string]entry `json:"schedule"`
}

const TeaTimeDuration = time.Minute*20

func durationToNextTea() time.Duration {
	t := time.Now()
	day := t.Day()
	hour := t.Hour()
	next_hour := ((hour+1)/2)*2 + 1
	if next_hour >= 24 {
		next_hour -= 24
		day += 1
	}
	next_tea := time.Date(t.Year(), t.Month(), day, next_hour, 0, 0, 0, time.Local)
	return next_tea.Sub(t)
}

func timeToNextTea() time.Time {
	return time.Time{}.Add(durationToNextTea())
}

func createTeaSchedule(Schedule map[string]entry) {
	//var schedule []time.Time

	for key, value := range Schedule {
		fmt.Println("Key:", key)
		for i, day := range value.Day {
			fmt.Printf("\t%d) Day: %s\n", i, day)
			for j, timez := range value.Time {
				fmt.Printf("\t\t%d) Time: %s\n", j, fmt.Sprintf("%s:00", timez))
				t,_ := time.Parse("15:04", timez)
				baseTime := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
				fmt.Println(t.Sub(baseTime))
				//schedule = append(schedule, )
			}
		}
	}
}

func main() {
	plan, _ := ioutil.ReadFile("bot.json")
	var teaTime bot_config
	e := json.Unmarshal(plan, &teaTime)
	if e != nil {
		log.Panic(e)
	}
	log.Println(teaTime)

	createTeaSchedule(teaTime.Schedule)

	bot, err := tgbotapi.NewBotAPI(teaTime.BotToken)
	if err != nil {
		log.Panic(err)
	}

	log.Printf("Tea group chat id is %x", teaTime.ChatId)

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
					msg := tgbotapi.NewMessage(teaTime.ChatId, "го чай")
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

		s := strings.Split(update.Message.Text, "@")
		if len(s) > 1 && s[1] != bot.Self.UserName {
			continue
		}

		bot_command := s[0]

		if bot_command == "/чай" || bot_command == "/tea" {
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
