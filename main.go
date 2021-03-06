package main

import (
	"encoding/json"
	"flag"
	"errors"
	"io/ioutil"
	"net/url"
	"net/http"
	"strings"
	"time"
	"fmt"
	"slackd/config"
	"github.com/mpdroog/beanstalkd" //"github.com/maxid/beanstalkd"
)

const ERR_WAIT_SEC = 5

func connect() (*beanstalkd.BeanstalkdClient, error) {
	queue, e := beanstalkd.Dial(config.C.Beanstalk)
	if e != nil {
		return nil, e
	}
	queue.Use("slack")
	if _, e := queue.Watch("slack"); e != nil {
		return nil, e
	}
	queue.Ignore("default")
	return queue, nil
}

func proc(m config.LineDiff) error {
	channel := "general"
	if len(m.Tags) > 0 {
		channel = m.Tags[0]
	}
	channel = "#" + channel

	str, e := json.Marshal(config.Webhook{
		Text: "",
		Channel: channel,
		Username: config.C.Username,
		IconEmoji: config.C.IconEmoji,
		Attachments: []config.WebhookAttachment{config.WebhookAttachment{
			Fallback: "File changed",
			Pretext: m.Hostname + ":" + m.Path,
			Text: m.Line,
			Fields: []config.WebhookAttachmentField{config.WebhookAttachmentField{
				Title: "Hostname",
				Value: m.Hostname,
				Short: true,
			}, config.WebhookAttachmentField{
				Title: "Date",
				Value: time.Now().Format("2006-Jan-02 15:04"),
				Short: true,
			}},
		}},
	})
	res, e := http.PostForm(
		config.C.Url, url.Values{"payload": {string(str)}},
	)
	if e != nil {
		return e
	}
	if res.StatusCode != 200 {
		defer res.Body.Close()
		txt, e := ioutil.ReadAll(res.Body)
		if e != nil {
			return e
		}
		return errors.New(fmt.Sprintf("HTTP=%d, txt=%s", res.StatusCode, string(txt)))
	}
	return nil
}

func main() {
	var configPath string
	flag.BoolVar(&config.Verbose, "v", false, "Show all that happens")
	flag.StringVar(&configPath, "c", "./config.json", "Configuration")
	flag.Parse()

	if e := config.Init(configPath); e != nil {
		panic(e)
	}
	if config.Verbose {
		fmt.Printf("%+v\n", config.C)
	}

	queue, e := connect()
	if e != nil {
		panic(e)
	}
	if config.Verbose {
		fmt.Println("SlackD(" + config.Hostname + ") slack-tube (ignoring default)")
	}

	for {
		job, e := queue.Reserve(0)
		if e != nil {
			fmt.Println("Beanstalkd err: " + e.Error())
			time.Sleep(time.Second * ERR_WAIT_SEC)
			if strings.HasSuffix(e.Error(), "broken pipe") {
				// Beanstalkd down, reconnect!
				q, e := connect()
				if e != nil {
					fmt.Println("Reconnect err: " + e.Error())
				}
				if q != nil {
					queue = q
				}
			}
			continue
		}
		if config.Verbose {
			fmt.Println(fmt.Sprintf("Parse job %d", job.Id))
			fmt.Println("JSON:\r\n" + string(job.Data))
		}

		var m config.LineDiff
		if e := json.Unmarshal(job.Data, &m); e != nil {
			panic(e)
		}

		if e := proc(m); e != nil {
			fmt.Println("WARN: Failed sending, retry in 20sec (msg=" + e.Error() + ")")
			continue
		}
		queue.Delete(job.Id)
		if config.Verbose {
			fmt.Println(fmt.Sprintf("Finished job %d", job.Id))
		}
	}
}