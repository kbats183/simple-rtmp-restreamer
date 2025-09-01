package apiserver

type AddTargetInfo struct {
	Name   string `json:"name"`
	Target string `json:"target"`
}

type DeleteTargetInfo struct {
	Target string `json:"target"`
}
