package model

type LeftJoin struct {
	Index string `json:"index"`
	Parent []string `json:"parent"`
	Child []string `json:"child"`
	Host string `json:"host"`
	Username string `json:"username"`
	Password string `json:"password"`
	Query map[string]interface{} `json:"query"`
	Size int `json:"size"`
	From int `json:"from"`
	Source interface{} `json:"_source"`
}


type TmpLeftJoin struct {
	Index string `json:"index"`
	Parent string `json:"parent"`
	Child string `json:"child"`
	Query map[string]interface{} `json:"query"`
	Host string `json:"host"`
	Username string `json:"username"`
	Password string `json:"password"`
	Size int `json:"size"`
	From int `json:"from"`
	Source interface{} `json:"_source"`
}



type LeftJoinFieldValues struct {
	Field string `json:"field"`
	Values []string `json:"values"`
}