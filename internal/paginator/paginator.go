package paginator

import (
	"fmt"
	"log"
	"questTracker/internal/model"
	"strconv"
	"strings"

	"github.com/go-telegram/bot/models"
)

const (
	ItemsPerPage   int = 3 // maybe set this in config
	callbackParams int = 4 // this too
)

func NewPaginator(offset int, items []model.Quest) []models.InlineKeyboardButton {
	var res []models.InlineKeyboardButton

	if offset < ItemsPerPage {
		res = append(res, models.InlineKeyboardButton{Text: "<", CallbackData: "skip"})
	} else {
		res = append(res, models.InlineKeyboardButton{Text: "<", CallbackData: fmt.Sprintf("pager prev 0 %d", offset-ItemsPerPage)})
	}

	for i := range ItemsPerPage {
		if len(items) > i {
			res = append(res, models.InlineKeyboardButton{Text: strconv.Itoa(items[i].ID), CallbackData: fmt.Sprintf("pager del %d %d", items[i].ID, offset)})
		} else {
			res = append(res, models.InlineKeyboardButton{Text: " ", CallbackData: "skip"})
		}
	}

	if len(items) == ItemsPerPage+1 {
		res = append(res, models.InlineKeyboardButton{Text: ">", CallbackData: fmt.Sprintf("pager next 0 %d", offset+ItemsPerPage)})
	} else {
		res = append(res, models.InlineKeyboardButton{Text: ">", CallbackData: "skip"})
	}

	return res
}

func ParseCallBack(data string) *model.PaginatorData {
	parsed := strings.Fields(data)
	if len(parsed) != callbackParams {
		return nil
	}

	if parsed[0] != "pager" {
		return nil
	}

	elem, err := strconv.Atoi(parsed[2])
	if err != nil {
		log.Println(err)
		return nil
	}

	offset, err := strconv.Atoi(parsed[3])
	if err != nil {
		log.Println(err)
		return nil
	}

	return &model.PaginatorData{Elem: elem, Finish: parsed[1] == "del", Offset: offset}
}
