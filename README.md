SlackD
==================
Write 'file diffs' to Slack through the webhook API.

We have written a small logging utility (deltareport) that reports
logfile changes and wanted to write these to our Slack chatscreen. This
application reads the Beanstalkd `slack`-tube, expects the `LineDiff`-struct (in JSON-format)
and forwards this data to the channel as defined in the first `Tag`.

> Use of this source code is governed by a BSD-style license that can be found in the LICENSE file.

```
Usage of ./slackd:
  -c="./config.json": Configuration
  -v=false: Show all that happens
```

LineDiff struct
```
type LineDiff struct {
	Hostname string
	Path     string
	Line     string
	Tags     []string
}
```

Ref
==================
* https://github.com/mpdroog/deltareport
* https://api.slack.com/incoming-webhooks
