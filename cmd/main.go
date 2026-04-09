package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	customErrors "questTracker/internal/errors"
	"questTracker/internal/model"
	myproxy "questTracker/internal/myProxy"
	"questTracker/internal/paginator"
	"regexp"
	"strconv"
	"strings"
	"time"

	_ "modernc.org/sqlite"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/joho/godotenv"
)

var (
	GreetingsMsg string
	db           *sql.DB
)

var ()

func main() {
	dbfake, err := sql.Open("sqlite", "file:notification.sqlite?cache=shared")
	if err != nil {
		log.Fatal(err)
	}
	db = dbfake

	_, err = db.Exec("CREATE TABLE IF NOT EXISTS notify(userID INTEGER, questID INTEGER, questName TEXT);")
	if err != nil {
		log.Fatal(err)
	}

	err = godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	opts := []bot.Option{
		bot.WithDefaultHandler(handler),
	}

	address, found := os.LookupEnv("HOST")
	if found {
		client, err := myproxy.MakeProxyClient(os.Getenv("PRNAME"), os.Getenv("PASSWORD"), address)
		if err != nil {
			log.Fatal("ErrMakeProxy: ", err)
		}
		opts = append(opts, bot.WithHTTPClient(10*time.Second, client))
	}

	token, found := os.LookupEnv("TOKEN")
	if !found {
		log.Fatal(customErrors.ErrFoundToken)
	}

	b, err := bot.New(token, opts...)
	if err != nil {
		log.Fatal("can't create bot instance: ", err)
	}
	log.Println("bot started")
	b.Start(ctx)
}

func handler(ctx context.Context, b *bot.Bot, update *models.Update) {
	switch {
	case update.Message != nil && update.Message.Text != "":
		slc := strings.Fields(update.Message.Text)
		if slc[0] == "/start" {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ReplyMarkup: models.InlineKeyboardMarkup{
					InlineKeyboard: [][]models.InlineKeyboardButton{{
						{Text: "Создать оповещение", CallbackData: "CreateNotification"},
						{Text: "Список оповещений", CallbackData: "pager next 0 0"}}},
				},
				ChatID: update.Message.Chat.ID,
				Text:   "Hello, " + update.Message.Chat.Username + GreetingsMsg,
			})
			return
		}

		ID, name, err := parseQuestID(update.Message.Text)
		if err != nil {
			fmt.Println(err)
			return
		}

		err = createNotify(ID, name, update.Message.From.ID)
		if err != nil {
			fmt.Println(err)
			return
		}

		_, err = b.SendMessage(ctx, &bot.SendMessageParams{
			ReplyMarkup: models.InlineKeyboardMarkup{
				InlineKeyboard: [][]models.InlineKeyboardButton{{
					{Text: "Создать ещё  одно оповещение", CallbackData: "CreateNotification"},
					{Text: "Список оповещений", CallbackData: "pager next 0 0"}}},
			},
			ChatID: update.Message.Chat.ID,
			Text:   fmt.Sprintf("Оповещение для задания \"%s\" создано\n(ID задания: %d)", name, ID),
		})
		if err != nil {
			errorHandle(ctx, b, update, err)
		}

	case update.CallbackQuery != nil:
		data := strings.Fields(update.CallbackQuery.Data)

		switch data[0] {
		case "CreateNotification":
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.CallbackQuery.From.ID,
				Text:   "Введите ID локального задания",
			})
		default:
			info := paginator.ParseCallBack(update.CallbackQuery.Data)
			if info == nil {
				break
			}
			if info.Finish {
				err := deleteElem(info.Elem)
				if err != nil {
					log.Println(err)
					errorHandle(ctx, b, update, err)
					break
				}
			}
			list, err := listNotify(update.CallbackQuery.From.ID, info.Offset)
			if err != nil {
				log.Println(err)
				errorHandle(ctx, b, update, err)
			}
			_, err = b.SendMessage(ctx, &bot.SendMessageParams{
				ReplyMarkup: models.InlineKeyboardMarkup{
					InlineKeyboard: [][]models.InlineKeyboardButton{{
						{Text: "Создать оповещение", CallbackData: "CreateNotification"},
						{Text: "Список оповещений", CallbackData: "NotificationList"}},
						paginator.NewPaginator(info.Offset, list),
					},
				},
				LinkPreviewOptions: &models.LinkPreviewOptions{IsDisabled: &[]bool{true}[0]}, //add to const
				ChatID:             update.CallbackQuery.From.ID,

				Text:      strings.Join(createURL(list[:min(paginator.ItemsPerPage, len(list))]), "\n"),
				ParseMode: models.ParseModeHTML,
			})
			if err != nil {
				log.Println(err)
			}
		}
	}
}

func parseQuestID(qID string) (int, string, error) {
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Timeout: 2 * time.Second,
	}

	ID, err := strconv.Atoi(qID)
	if err != nil {
		re, err := regexp.Compile(`quest=(\d+)`)
		if err != nil {
			log.Println(err)
			return 0, "", err
		}

		res := re.FindSubmatch([]byte(qID))
		ID, err = strconv.Atoi(string(res[1]))
		if err != nil {
			return 0, "", err
		}
	}

	resp, err := client.Head("https://www.wowhead.com/ru/quest=" + strconv.Itoa(ID))
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()

	re, err := regexp.Compile(fmt.Sprintf("quest=%d/(.*)", ID))
	if err != nil {
		log.Println(err)
		return 0, "", err
	}
	name := re.FindSubmatch([]byte(resp.Header.Get("Location")))

	if resp.StatusCode != http.StatusOK && strings.Contains(resp.Header.Get("Location"), "/quests?notFound=") {
		log.Println(err)
		return 0, "", errors.New("Quest doesn't exist. It may have been removed from the game.")
	}
	return ID, strings.ReplaceAll(string(name[1]), "-", " "), nil
}

func createNotify(qID int, qName string, uID int64) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := db.ExecContext(ctx, "INSERT INTO notify VALUES($1, $2, $3)", uID, qID, qName)
	if err != nil {
		return err
	}
	return nil
}

func listNotify(uID int64, offset int) ([]model.Quest, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	slc, err := db.QueryContext(ctx, "SELECT questID, questName FROM notify WHERE userID = $1 ORDER BY questID LIMIT $2 OFFSET $3", uID, paginator.ItemsPerPage+1, offset)
	if err != nil {
		log.Printf("can't execute \"SELECT questID, questName FROM notify WHERE userID = %d ORDER BY questID LIMIT %d OFFSET %d ", uID, paginator.ItemsPerPage+1, offset)
		return nil, customErrors.ErrGetNotifyList
	}
	defer slc.Close()

	res := []model.Quest{}
	for slc.Next() {
		res = append(res, model.Quest{})
		err := slc.Scan(&res[len(res)-1].ID, &res[len(res)-1].Name)
		if err != nil {
			log.Println(err)
		}
	}

	return res, nil
}

func createURL(qIDs []model.Quest) []string {
	res := []string{}
	for _, v := range qIDs {
		res = append(res, fmt.Sprintf("<a href='https://www.wowhead.com/ru/quest=%d'>%s (ID: %d)</a>", v.ID, v.Name, v.ID))
	}
	return res
}

func errorHandle(ctx context.Context, b *bot.Bot, update *models.Update, err error) {
	switch {
	case errors.Is(err, customErrors.ErrGetNotifyList):
	}
	b.SendMessage(ctx, &bot.SendMessageParams{
		ReplyMarkup: models.InlineKeyboardMarkup{
			InlineKeyboard: [][]models.InlineKeyboardButton{{
				{Text: "Создать оповещение", CallbackData: "CreateNotification"},
				{Text: "Список оповещений", CallbackData: "NotificationList"}}},
		},
		ChatID: update.Message.Chat.ID,
		Text:   "Hello, " + update.Message.Chat.Username + GreetingsMsg,
	})
}

func deleteElem(qID int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	slc, err := db.QueryContext(ctx, "DELETE FROM notify WHERE questID = $1", qID)
	if err != nil {
		return err
	}
	defer slc.Close()

	return nil
}
