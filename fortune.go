package main

import (
	"fmt"
	"github.com/kballard/goirc/irc"
	"log"
	"os/exec"
	"strings"
)

func init() {
	RegisterModule("fortune", func() Module {
		return &FortuneMod{}
	})
}

type FortuneMod struct {
	cmd []string
}

func (f *FortuneMod) Init(b *Bot, conn irc.SafeConn) error {
	conf := b.Config.Search("mod", "fortune")
	theo := conf.Search("theo")
	f.cmd = []string{"9", "fortune"}

	conn.AddHandler("PRIVMSG", func(c *irc.Conn, l irc.Line) {
		if l.Args[1] == ".fortune" {
			strs := fixup(f.fortune(""))
			log.Printf("fortune %+v", strs)
			for _, s := range strs {
				if l.Args[0][0] == '#' {
					c.Privmsg(l.Args[0], s)
				} else {
					c.Privmsg(l.Src.String(), s)
				}
			}
		}
		if l.Args[1] == ".theo" {
			strs := fixup(f.fortune(theo))
			log.Printf("theo %+v", strs)
			for _, s := range strs {
				if l.Args[0][0] == '#' {
					c.Privmsg(l.Args[0], s)
				} else {
					c.Privmsg(l.Src.String(), s)
				}
			}
		}
		if l.Args[1] == ".bullshit" {
			var strs []string
			out, err := exec.Command("9", "bullshit").CombinedOutput()
			if err != nil {
				strs = []string{err.Error()}
			} else {
				strs = fixup(string(out))
			}
			log.Printf("bullshit %+v", strs)
			for _, s := range strs {
				if l.Args[0][0] == '#' {
					c.Privmsg(l.Args[0], s)
				} else {
					c.Privmsg(l.Src.String(), s)
				}
			}
		}
	})

	log.Printf("forutne module initialized with cmd %s", strings.Join(f.cmd, " "))
	return nil
}

func (f *FortuneMod) fortune(file string) string {
	cmd := f.cmd

	if file != "" {
		cmd = append(cmd, file)
	}
	c := exec.Command(cmd[0], cmd[1:]...)

	out, err := c.CombinedOutput()

	if err != nil {
		return fmt.Sprintf("fortune error: %S", err)
	} else {
		return string(out)
	}
}

func (m *FortuneMod) Reload() error {
	return nil
}

func (m *FortuneMod) Call(args ...string) error {
	return nil
}

func fixup(f string) []string {
	str := strings.Replace(f, "\t", " ", -1)
	strs := strings.Split(str, "\n")

	return strs
}
