package main

import (
	"gopkg.in/telegram-bot-api.v4"
	"fmt"
	"log"
	"time"
	"sort"
	"strings"
	"io/ioutil"
	"encoding/json"
)

type entry struct {
	Day   []string `json:"day"`
	Time  []string `json:"time"`
}

type bot_config struct {
	BotToken     string           `json:"tea_time_bot_token"`
	ChatId       int64            `json:"tea_time_chat_id"`
	TeaDuration  int              `json:"tea_time_duration"`
	Schedule     map[string]entry `json:"schedule"`
}

type TeaTimes []time.Time

func (s TeaTimes) Len() int {
	return len(s)
}

func (s TeaTimes) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s TeaTimes) Less(i, j int) bool {
	return s[i].Before(s[j])
}

func durationToNextTea(nextTea time.Time) time.Duration {
	t := time.Now()
	return nextTea.Sub(t)
}

func timeToNextTea(nextTea time.Time) time.Time {
	return time.Time{}.Add(durationToNextTea(nextTea))
}

func durationFromWeekStart(day string) time.Duration {
	switch day {
		case "monday", "mon":
			return time.Hour * 24 * 0
		case "tuesday", "tue":
			return time.Hour * 24 * 1
		case "wednesday", "wed":
			return time.Hour * 24 * 2
		case "thursday", "thu":
			return time.Hour * 24 * 3
		case "friday", "fri":
			return time.Hour * 24 * 4
		case "saturday", "sat":
			return time.Hour * 24 * 5
		case "sunday", "sun":
			return time.Hour * 24 * 6
	}

	return 0
}

func bod(t time.Time) time.Time {
	year, month, day := t.Date()

	return time.Date(year, month, day, 0, 0, 0, 0, t.Location())
}

func getNearestMonday(t time.Time) time.Time {
	for {
		if t.Weekday() != time.Monday {
			t = t.AddDate(0, 0, -1)
		} else {
			return t
		}
	}
}

func createTeaSchedule(startTime time.Time, schedule map[string]entry) []time.Time {
	var items TeaTimes

	current := startTime
	monday := bod(getNearestMonday(current))

	/* we need at least 14 items */
	for len(items) == 0 {
		for key, value := range schedule {
			fmt.Println("Key:", key)

			for i, day := range value.Day {
				fmt.Printf("\t%d) Day: %s\n", i, day)

				for j, timez := range value.Time {
					t,e := time.Parse("15:04", timez)
					if e != nil {
						continue
					}
					scheduledTime := monday.Add(t.Sub(bod(t)) + durationFromWeekStart(day))
					fmt.Printf("\t\t%d) Time: %s\n", j, fmt.Sprintf("%s:00", timez))
					fmt.Println( scheduledTime )
					fmt.Println( scheduledTime.Sub(current) )
					if scheduledTime.Sub(current) > 0 {
						items = append(items, scheduledTime)
					}
				}
			}
		}

		monday = monday.AddDate(0, 0, 7)
	}

	sort.Sort(items)
	fmt.Println( items )
	return items
}

func main() {
	plan, _ := ioutil.ReadFile("bot.json")
	var teaTime bot_config
	e := json.Unmarshal(plan, &teaTime)
	if e != nil {
		log.Panic(e)
	}
	log.Println(teaTime)

	teaSchedule := createTeaSchedule(time.Now(), teaTime.Schedule)
	teaScheduleIndex := 0

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
			toTea := durationToNextTea(teaSchedule[teaScheduleIndex])
			log.Printf("to next tea: %d", toTea)

			fmt.Println("Tick at", t)

			if ( need_create_timer_to_send_alarm ) {
				need_create_timer_to_send_alarm = false
				timer := time.NewTimer(toTea)
				go func() {
					<- timer.C
					msg := tgbotapi.NewMessage(teaTime.ChatId, "го чай")
					bot.Send(msg)
					teaScheduleIndex += 1
					if teaScheduleIndex == len(teaSchedule) {
						teaSchedule = createTeaSchedule(time.Now(), teaTime.Schedule)
						teaScheduleIndex = 0
					}
					need_create_timer_to_send_alarm = true

					tea_time_ongoing = true
					tea_ongoin := time.NewTimer(time.Minute*time.Duration(teaTime.TeaDuration))
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
				log.Printf("next tea at: %s\n", teaSchedule[teaScheduleIndex])
				msg_txt = fmt.Sprintf("До чая осталось: %s.", timeToNextTea(teaSchedule[teaScheduleIndex]).Format("15:04:05"))
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
