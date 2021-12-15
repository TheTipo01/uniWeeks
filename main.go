package main

import (
	"database/sql"
	"github.com/bwmarrin/lit"
	_ "github.com/go-sql-driver/mysql"
	"github.com/kkyr/fig"
	"github.com/robfig/cron/v3"
	tb "gopkg.in/tucnak/telebot.v2"
	"strconv"
	"strings"
	"time"
)

// Config holds data parsed from the config.yml
type Config struct {
	Token         string `fig:"token" validate:"required"`
	DSN           string `fig:"database" validate:"required"`
	LogLevel      string `fig:"loglevel" validate:"required"`
	Notifications bool   `fig:"notifications"`
}

var (
	token         string
	db            *sql.DB
	cache         = make(map[int64]bool)
	notifications bool
)

func init() {
	lit.LogLevel = lit.LogError

	var cfg Config
	err := fig.Load(&cfg, fig.File("config.yml"))
	if err != nil {
		lit.Error(err.Error())
		return
	}

	notifications = cfg.Notifications
	token = cfg.Token

	// Set lit.LogLevel to the given value
	switch strings.ToLower(cfg.LogLevel) {
	case "logwarning", "warning":
		lit.LogLevel = lit.LogWarning

	case "loginformational", "informational":
		lit.LogLevel = lit.LogInformational

	case "logdebug", "debug":
		lit.LogLevel = lit.LogDebug
	}

	// Open database connection
	db, err = sql.Open("mysql", cfg.DSN)
	if err != nil {
		lit.Error("Error opening db connection, %s", err)
		return
	}

	// Creates table used to store everything
	execQuery(tblUsers)
	loadCache()
}

func main() {
	// Create bot
	b, err := tb.NewBot(tb.Settings{
		Token:  token,
		Poller: &tb.LongPoller{Timeout: 10 * time.Second},
	})
	if err != nil {
		lit.Error(err.Error())
		return
	}

	// Keyboard things
	var (
		// Universal markup builders.
		menu = &tb.ReplyMarkup{ResizeReplyKeyboard: true, OneTimeKeyboard: true}

		// Reply buttons.
		btnEven = menu.Text("Pari")
		btnOdd  = menu.Text("Dispari")
	)

	menu.Reply(
		menu.Row(btnEven),
		menu.Row(btnOdd),
	)

	// /start command
	b.Handle("/start", func(m *tb.Message) {
		_, _ = b.Send(m.Sender, "Ciao! Usa la tastiera qui sotto per configurare se sei matricola pari o dispari, così ti potrò avvisare ogni domenica pomeriggio se puoi andare in presenza o no!", menu)
	})

	// Buttons
	b.Handle(&btnEven, func(m *tb.Message) {
		_, _ = b.Send(m.Sender, "Perfetto! Ti avvisero ogni settimana pari!\nSe vuoi cambiare selezione, riapri la tastiera\n\nRicordati che puoi usare il comando /quando per sapere se questa settimana puoi andare o no")
		updateDB(m.Sender.ID, true)
	})

	b.Handle(&btnOdd, func(m *tb.Message) {
		_, _ = b.Send(m.Sender, "Perfetto! Ti avvisero ogni settimana dispari!\nSe vuoi cambiare selezione, riapri la tastiera\n\nRicordati che puoi usare il comando /quando per sapere se questa settimana puoi andare o no")
		updateDB(m.Sender.ID, false)
	})

	// /disabilita command to delete the user from the DB
	b.Handle("/disabilita", func(m *tb.Message) {
		_, _ = b.Send(m.Sender, "Non ti inviero più messaggi!")
		deleteFromDB(m.Sender.ID)
	})

	// /quando command to know if in the current week you can go to lessons physically
	b.Handle("/quando", func(m *tb.Message) {
		if val, ok := cache[m.Sender.ID]; ok {
			// User exists in cache
			_, week := time.Now().ISOWeek()
			weekEven := week%2 == 0
			strWeekEven := strconv.Itoa(week)

			_, _ = b.Send(m.Sender, createMessage(val, weekEven, strWeekEven, "Questa settimana"), tb.ModeMarkdown)
		} else {
			_, _ = b.Send(m.Sender, "Non hai ancora configurato se sei pari o dispari! Usa la tastiera qui sotto per farlo")
		}
	})

	if notifications {
		lit.Info("Notifications are enabled!")

		// Cronjob, to send messages every sunday afternoon
		loc, _ := time.LoadLocation("Europe/Rome")
		c := cron.New(cron.WithLocation(loc))
		// At 18:00 on Saturday
		_, _ = c.AddFunc("0 18 * * 6", func() {
			// Calculate the current week
			_, week := time.Now().Add(time.Hour * 168).ISOWeek()
			weekEven := week%2 == 0
			strWeekEven := strconv.Itoa(week)

			// Iterate every user
			for id, userEven := range cache {
				user := &tb.User{ID: id}
				_, _ = b.Send(user, createMessage(userEven, weekEven, strWeekEven, "La settimana prossima"), tb.ModeMarkdown)
			}
		})
		c.Start()
	} else {
		lit.Info("Notifications are disabled!")
	}

	// Starts the bot
	lit.Info("uniWeeks is now running")
	b.Start()
}

func createMessage(userEven bool, weekEven bool, strWeekEven string, base string) string {
	out := base

	if userEven {
		if weekEven {
			out += " "
		} else {
			// User is even and the week is odd
			out += " NON "
		}
	} else {
		if weekEven {
			// User is odd and the week is odd
			out += " NON "
		} else {
			out += " "
		}
	}

	out += "puoi andare in presenza, perchè è la numero " + strWeekEven + "\nRicordati di prenotare le lezioni su [Student Booking](https://unito.sbk.cineca.it/)"

	return out
}
