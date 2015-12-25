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

var readonly bool
const ERR_WAIT_SEC = 5

func connect() (*beanstalkd.BeanstalkdClient, error) {
	queue, e := beanstalkd.Dial(config.C.Beanstalk)
	if e != nil {
		return nil, e
	}
	// Only listen to email queue.
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
	res, e := http.PostForm(
		config.C.Url,
		url.Values{
			"payload": {m.Line},
			"channel": {channel},
			"username": {config.C.Username},
			"icon_emoji": {config.C.IconEmoji},
		},
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
		return errors.New("HTTP != 200, txt=" + string(txt))
	}
	return nil
}

func main() {
	var configPath string
	flag.BoolVar(&config.Verbose, "v", false, "Show all that happens")
	flag.StringVar(&configPath, "c", "./config.json", "Configuration")
	flag.BoolVar(&readonly, "r", false, "Don't email but flush to stdout")
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
	if readonly {
		fmt.Println("!! ReadOnly mode !!")
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
			// TODO: Isolate deverr from senderr
			// Processing trouble?
			fmt.Println("WARN: Failed sending, retry in 20sec (msg=" + e.Error() + ")")
			continue
		}
		queue.Delete(job.Id)
		if config.Verbose {
			fmt.Println(fmt.Sprintf("Finished job %d", job.Id))
		}
	}
}