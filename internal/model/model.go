package model

type WowHeadJson struct {
	Info []WowHeadQuest `json:"data"`
}

type WowHeadQuest struct {
	ID     int   `json:"id"`
	Ending int64 `json:"ending"`
}

type Quest struct {
	ID   int    `db:"questID"`
	Name string `db:"questName"`
}

type PaginatorData struct {
	Offset int
	Elem   int
	Finish bool
}

var Addons = []string{"legion", "bfa", "sl", "df", "tww", "mn"}
