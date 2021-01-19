package model

type LeftJoin struct {
	Index string `json:"index"`
	Parent []string `json:"parent"`
	Child []string `json:"child"`
	Query map[string]interface{} `json:"query"`
	Size int `json:"size"`
	From int `json:"from"`
	Source interface{} `json:"_source"`
}


type LeftJoinFieldValues struct {
	Field string `json:"field"`
	Values []string `json:"values"`
}