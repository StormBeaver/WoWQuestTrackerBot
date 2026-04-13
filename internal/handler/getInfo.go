package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"questTracker/internal/model"
	"regexp"
	"time"
)

func GetInfo(addon string, resultMap map[int]time.Time) (map[int]time.Time, error) {
	resp, err := http.Get("https://www.wowhead.com/world-quests/" + addon + "/eu")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	re, err := regexp.Compile(`Listview\((.*)\)`)
	if err != nil {
		return nil, err
	}

	jsn := re.FindSubmatch(body)[1]
	res := &model.WowHeadJson{}

	err = json.Unmarshal(jsn, res)
	if err != nil {
		return nil, err
	}

	for _, v := range res.Info {
		resultMap[v.ID] = time.UnixMilli(v.Ending)
	}

	return resultMap, nil
}
