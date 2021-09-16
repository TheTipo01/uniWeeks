package main

import (
	"database/sql"
	"github.com/bwmarrin/lit"
	_ "github.com/go-sql-driver/mysql"
	"github.com/robfig/cron/v3"
	"github.com/spf13/viper"
	tb "gopkg.in/tucnak/telebot.v2"
	"strconv"
	"strings"
	"time"
)

var (
	token string
	db    *sql.DB
)

func init() {
	lit.LogLevel = lit.LogError

	viper.SetConfigName("config")
	viper.SetConfigType("yml")
	viper.AddConfigPath(".")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found
			lit.Error("Config file not found! See example_config.yml")
			return
		}
	} else {
		// Config file found
		token = viper.GetString("token")

		// Set lit.LogLevel to the given value
		switch strings.ToLower(viper.GetString("loglevel")) {
		case "logwarning", "warning":
			lit.LogLevel = lit.LogWarning

		case "loginformational", "informational":
			lit.LogLevel = lit.LogInformational

		case "logdebug", "debug":
			lit.LogLevel = lit.LogDebug
		}

		// Open database connection
		db, err = sql.Open("mysql", viper.GetString("database"))
		if err != nil {
			lit.Error("Error opening db connection, %s", err)
			return
		}
	}
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
		var userEven bool
		err := db.QueryRow("SELECT even FROM users WHERE id = ?", m.Sender.ID).Scan(&userEven)
		if err != nil {
			_, _ = b.Send(m.Sender, "Non hai ancora configurato se sei pari o dispari! Usa la tastiera qui sotto per farlo")
			return
		}

		_, week := time.Now().ISOWeek()
		weekEven := week%2 == 0
		strWeekEven := strconv.Itoa(week)

		_, _ = b.Send(m.Sender, createMessage(userEven, weekEven, strWeekEven, "Questa settimana"))
	})

	// Cronjob, to send messages every sunday afternoon
	loc, err := time.LoadLocation("Europe/Rome")
	c := cron.New(cron.WithLocation(loc))
	_, _ = c.AddFunc("0 18 * * 0", func() {
		rows, err := db.Query("SELECT id, even FROM users")
		if err != nil {
			lit.Error("Error while querying db for weekly message: " + err.Error())
			return
		}

		// Calculate the current week
		_, week := time.Now().Add(time.Hour * 168).ISOWeek()
		weekEven := week%2 == 0
		strWeekEven := strconv.Itoa(week)

		// Iterate every user
		for rows.Next() {
			var (
				id       int
				userEven bool
			)
			_ = rows.Scan(&id, &userEven)

			user := &tb.User{ID: id}
			_, _ = b.Send(user, createMessage(userEven, weekEven, strWeekEven, "Da domani"))
		}

		_ = rows.Close()
	})
	c.Start()

	// Starts thg bot
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

	out += "puoi andare in presenza, perchè è la settimana numero " + strWeekEven + "\nRicordati di prenotare le lezioni su Student Booking"

	return out
}
