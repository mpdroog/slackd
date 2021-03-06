package config

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
)

var (
	C *Conf
	Log *log.Logger
	Verbose bool
	Hostname string
)

type Conf struct {
	Url       string
	Username  string
	IconEmoji string
	Beanstalk string
}

type LineDiff struct {
	Hostname string
	Path     string
	Line     string
	Tags     []string
}

type WebhookAttachmentField struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool `json:"short"`
}
type WebhookAttachment struct {
	Fallback string `json:"fallback"`
	Pretext string `json:"pretext"`
	Text string `json:"text"`
	Fields []WebhookAttachmentField `json:"fields"`
}
type Webhook struct {
	Channel string   `json:"channel"`
	Username  string `json:"username"`
	IconEmoji string `json:"icon_emoji"`
	Text string      `json:"text"`
	Attachments []WebhookAttachment `json:"attachments"`
}

func Init(path string) error {
	b, e := ioutil.ReadFile(path)
	if e != nil {
		return e
	}
	C = new(Conf)
	if e := json.Unmarshal(b, C); e != nil {
		return e
	}
	Hostname, e = os.Hostname()
	if e != nil {
		return e
	}

	Log = log.New(os.Stdout, "slackd ", log.LstdFlags)
	return nil
}