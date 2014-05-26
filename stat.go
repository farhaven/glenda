package main

import (
	"database/sql"
	_"github.com/mattn/go-sqlite3"

	"github.com/kballard/goirc/irc"

	"log"

	"time"
	"bufio"
	"strings"
	"unicode/utf8"
	"unicode"
	"fmt"
)

type StatMod struct {
	db *sql.DB
}

var (
	tables = []string{
		`CREATE TABLE IF NOT EXISTS
			Stat (
				id INTEGER PRIMARY KEY autoincrement,
				ident TEXT,

				chars     INTEGER DEFAULT 0,
				words     INTEGER DEFAULT 0,
				sentences INTEGER DEFAULT 0,
				actions   INTEGER DEFAULT 0,
				lines     INTEGER DEFAULT 1,
				last      INTEGER NOT NULL,
				active    DECIMAL NOT NULL
			)`,
	}
)

func init () {
	RegisterModule("stat", func () Module {
		return &StatMod{}
	})
}

func (m *StatMod) Init (b *Bot, conn irc.SafeConn) (err error) {
	conf := b.Config.Search("mod", "stat")

	m.db, err = sql.Open("sqlite3", conf.Search("path"))

	if err != nil {
		log.Printf("stat module failed to open %q: %s\n", conf.Search("path"), err)
		return
	}

	for _, t := range tables {
		_, err = m.db.Exec(t)

		if err != nil {
			log.Printf("stat module failed to create table: %s\n%q\n", err, t)
			return
		}
	}

	conn.AddHandler("PRIVMSG", func (c *irc.Conn, l irc.Line) {
		args := strings.Split(l.Args[1], " ")

		if len(args) >= 1 {
			if args[0] == ".stat" {

				var recipient string

				if l.Args[0][0] == '#'{
					recipient = l.Args[0]
				} else {
					recipient = l.Src.String()
				}

				if len(args) >= 2 {
					s, err := m.stat(args[1])

					if err != nil {
						log.Printf("stat failed for %q: %s", args[1], err)
						return
					}

					if s == nil {
						c.Privmsg(recipient, `no such user: "` + args[1] + `"`)
						return
					}
					
					c.Privmsg(recipient, s.String())
				} else {
					c.Privmsg(recipient, "missing argument")
				}
			} else {
				if err := m.update(l.Src.String(), l.Args[1]); err != nil {
					log.Printf("update failed for %q, %q: %s",
						l.Src.String(),
						l.Args[1:],
						err,
					)
				}
			}
		}
	})

	conn.AddHandler(irc.ACTION, func (c *irc.Conn, l irc.Line) {
		if err := m.action(l.Src.String()); err != nil {
			log.Printf("action update failed for %q: %s",
				l.Src.String(),
				err,
			)
		}
	})

	log.Println("stat module initialised")
	return nil
}

func (m *StatMod) Reload () error {
	return nil
}

func (m *StatMod) Call (args ...string) error {
	return nil
}

func (m *StatMod) action (ident string) error {
	res, err := m.db.Exec(`
		UPDATE
			Stat
		SET
			actions = actions + 1,
			lines = lines + 1,
			active = active + (strftime('%H', 'now', 'localtime') - active) / (lines + 1)
		WHERE
			ident = ?`,
		ident,
	)

	if err != nil {
		return err
	}

	n, err := res.RowsAffected();

	if err != nil {
		return err
	}

	if n < 1 {
		_, err = m.db.Exec(`
			INSERT INTO
				Stat (id, ident, actions, lines, last, active)
			VALUES
				(null, ?, 1, 1, strftime('%s', 'now', 'localtime'), strftime('%H', 'now', 'localtime'))
			`,
			ident,
		)

		if err != nil {
			return err
		}
	}

	return nil
}

func (m *StatMod) update (ident, message string) error {
	
	chars := len(message)
	words := Nwords(message)
	sentences := Nsentences(message)

	res, err := m.db.Exec(`
		UPDATE
			Stat
		SET
			chars = chars + ?,
			words = words + ?,
			sentences = sentences + ?,
			lines = lines + 1,
			last = strftime('%s', 'now'),
			active = active + (strftime('%H', 'now', 'localtime') - active) / (lines + 1)
		WHERE
			ident = ?`,
		chars,
		words,
		sentences,
		ident,
	)

	if err != nil {
		return err
	}

	n, err := res.RowsAffected();

	if err != nil {
		return err
	}

	if n < 1 {
		_, err = m.db.Exec(`
			INSERT INTO
				Stat (id, ident, chars, words, sentences, lines, last, active)
			VALUES
				(null, ?, ?, ?, ?, 1, strftime('%s', 'now', 'localtime'), strftime('%H', 'now', 'localtime'))`,
			ident, chars, words, sentences,
		)

		if err != nil {
			return err
		}
	}

	return nil
}

func Nwords (text string) int64 {
	s := bufio.NewScanner(strings.NewReader(text))
	s.Split(bufio.ScanWords)
	var count int64
	for s.Scan() {
		count++
	}
	return count
}

func Nsentences (text string) int64 {
	s := bufio.NewScanner(strings.NewReader(text))

	s.Split(func (data []byte, atEOF bool) (advance int, token []byte, err error) {
		start := 0
		width := 0
		for ; start < len(data); start += width {
			var r rune

			r, width = utf8.DecodeRune(data[start:])

			if !unicode.Is(unicode.STerm, r) {
				break
			}
		}

		if atEOF && len(data) == 0 {
			return 0, nil, nil
		}

		for i := 0; i < len(data); i += width {
			var r rune

			r, width = utf8.DecodeRune(data[i:])

			if unicode.Is(unicode.STerm, r) {
				return i + width, data[start:i], nil
			}
		}

		if atEOF && len(data) > start {
			return len(data), data[start:], nil
		}

		return 0, nil, nil
	})

	var count int64

	for s.Scan() {
		count++
	}

	return count
}

type Stat struct {
	Ident string
	Chars, Words, Sentences, Actions, Lines int64
	Last time.Time
	Active int64
}

func (s Stat) String () string {
	return fmt.Sprintf(
		"%s - %d chars, %d words, %d sentences, %d actions, %d lines, " +
		"last message at %v, most active at %02d00",
		s.Ident, s.Chars, s.Words, s.Sentences, s.Actions, s.Lines,
		s.Last, s.Active,
	)
}

func (m *StatMod) stat (ident string) (*Stat, error) {
	var (
		s Stat
		last int64
	)

	err := m.db.QueryRow(`
		SELECT
			chars, words, sentences, actions, lines, last, CAST(active AS INTEGER)
		FROM
			Stat
		WHERE
			ident = ?`,
		ident,
	).Scan(
		&s.Chars, &s.Words, &s.Sentences, &s.Actions, &s.Lines, &last, &s.Active,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	
	s.Last = time.Unix(last, 0)

	s.Ident = ident

	return &s, nil
}
