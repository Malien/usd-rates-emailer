# USD-to-UAH rates emailer

_Ah yes. Yet another one of those.._

![image](https://github.com/Malien/usd-rates-emailer/assets/7205038/3bee37a7-18b3-40ed-a094-d7a08929d023)

### Startup

Plain and simple: `go run ./cmd/server`

In order to publish the newsletter to all of the subscribers: `go run ./cmd/mailer`

Wanna run tests? `go test` :+1:

All of the configuration is in `conf/` director. Be careful! It is not a complete configuration. It is missing SMTP server authentication secrets. One can simply add `conf/config.local.toml` to add private overrides akin to:
```toml
[email]
username = "<smtp usernal>"
password = "<smtp password>"
from = "<which email address messages are sent from>"

[email.smpt]
host = "<smtp host>"
port = 465
ssl = true
```
Or just provide a couple of environment variables:
- `EMAIL_USERNAME`
- `EMAIL_PASSWORD`
- `EMAIL_FROM`
- `EMAIL_SMTP` — SMTP server host

Wanna run as a docker image: `docker-compose up` to all your heart's content. Be warned that for the email sendouts to work, those four environment variables has to be set.

### Tech choices
If there is one thing that permiates the project, than it is: "Just keep it simple stupid!".

- The database is the simplest thing that qualifies as one. Aka `sqlite`. Just point it to a file, turn on WAL mode, and you are off to the races! No extra processes. No extra containers. No network calls. Just stupidly fast queries. **Bonus points**: it is super easy to write and run tests against a in-memory sqlite database (aka. `filename = :memory:`). One downside of using sqlite, is that it requires CGO. Yeaaaah, cgo is rough. The compile times took a big 'ol hit. 
- Since "migrations" are a required part of the task, the mechanism is there. No extra deps, just a `create table if not exists migrations`, and an array of migrations strings just stored gloablly in `migration.go`. Nothing fancy.
- Just a straight up `net/http` http server. Thanks go v1.22 for the new `http.ServeMux` patterns.
- Just a straight up `log` logger. Although I'm not particularly fond of it, it suffices for the usecase. I do like `zerolog` though.
- Just a straight up `database/sql` querries. Simple SQL. Simple oprations. Beautiful. None of those pesky leaky ORM abstractions. Yugh.
- Cron job scheduling is just plain 'ol alpine's `crond`. One could argue systemd services are a better fit. Maybe. Maybe. Alpine doesn't do systemd + cron is simple enough. Since the use of cron, the application is split into two entrypoints (hence the `cmd/server` and `cmd/mailer`). One writes subscribers into sqlite database. Another reads them, and sends emails. I guess could've `time.Sleep` inside of the goroutine, but where's the fun in that?
- Want to send an email? Let's travel back to 1970s and SMTP our way out of this!
- I do like `spf13/viper` for dealing with configs. It is nice and covers all of my needs for nierarhical config file + ENV variable support.
- I guess there is also a dep on `matoous/go-nanoid/v2`, just a a random requestId string to thread through possibly interleaving loger calls.
- No pesky mocks in tests. Well... I do mock calls to the exchange rates API... Well and to the SMTP server as well... But other than those: real database, real modules, real http layers. And those are external services. It would've been inconvenient to recieve multiple emails every time someone wants to run a test.
