package model

type LeftJoin struct {
	Index string `json:"index"`
	Parent string `json:"parent"`
	Child string `json:"child"`
	Query map[string]interface{} `json:"query"`
}
