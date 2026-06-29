package events

import "github.com/linkease/quick-start/istore-backend/models"

func ShouldTriggerIfaceEvent(evt models.DHNSChangeRequest) bool {
	switch evt.Action {
	case "ifaceEvent":
		return len(evt.Params) == 2 && (evt.Params[0] == "up" || evt.Params[0] == "down")
	case "uciChange":
		return true
	default:
		return false
	}
}
