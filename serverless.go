package telebot

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// A Webhook configures the poller for webhooks. It opens a port on the given
// listen adress. If TLS is filled, the listener will use the key and cert to open
// a secure port. Otherwise it will use plain HTTP.
// If you have a loadbalancer ore other infrastructure in front of your service, you
// must fill the Endpoint structure so this poller will send this data to telegram. If
// you leave these values empty, your local adress will be sent to telegram which is mostly
// not what you want (at least while developing). If you have a single instance of your
// bot you should consider to use the LongPoller instead of a WebHook.
// You can also leave the Listen field empty. In this case it is up to the caller to
// add the Webhook to a http-mux.
type ServerlessWebhook struct {
	Listen   string
	Endpoint *WebhookEndpoint
	Request  *http.Request
	Response http.ResponseWriter
	dest     chan<- Update
	bot      *Bot
}

func (h *ServerlessWebhook) getFiles() map[string]File {
	m := make(map[string]File)
	return m
}

func (h *ServerlessWebhook) getParams() map[string]string {
	param := make(map[string]string)
	if h.Endpoint != nil {
		param["url"] = h.Endpoint.PublicURL
	}
	return param
}

func (h *ServerlessWebhook) Poll(b *Bot, dest chan Update, stop chan struct{}) {
	res, err := b.sendFiles("setWebhook", h.getFiles(), h.getParams())
	if err != nil {
		b.debug(fmt.Errorf("setWebhook failed %q: %v", string(res), err))
		close(stop)
		return
	}
	var result registerResult
	err = json.Unmarshal(res, &result)
	if err != nil {
		b.debug(fmt.Errorf("bad json data %q: %v", string(res), err))
		close(stop)
		return
	}
	if !result.Ok {
		b.debug(fmt.Errorf("cannot register webhook: %s", result.Description))
		close(stop)
		return
	}

	var update Update
	err = json.NewDecoder(h.Request.Body).Decode(&update)
	if err != nil {
		h.bot.debug(fmt.Errorf("cannot decode update: %v", err))
		return
	}
	dest <- update
	close(stop)
	return
}

func (h *ServerlessWebhook) waitForStop(stop chan struct{}) {
	<-stop
	close(stop)
}

// The handler simply reads the update from the body of the requests
// and writes them to the update channel.
func (h *ServerlessWebhook) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var update Update
	err := json.NewDecoder(r.Body).Decode(&update)
	if err != nil {
		h.bot.debug(fmt.Errorf("cannot decode update: %v", err))
		return
	}
	h.dest <- update
}
