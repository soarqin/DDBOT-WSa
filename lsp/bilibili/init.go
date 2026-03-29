package bilibili

import (
	"github.com/cnxysoft/DDBOT-WSa/lsp/concern"
	"time"
)

func init() {
	concern.RegisterConcern(NewConcern(concern.GetNotifyChan()))
	refreshCookieJar()
	refreshNavWbi()
	go func() {
		ticker := time.NewTicker(time.Minute * 60)
		defer ticker.Stop()
		for range ticker.C {
			refreshCookieJar()
		}
	}()
	go func() {
		ticker := time.NewTicker(2 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			refreshNavWbi()
		}
	}()
}
