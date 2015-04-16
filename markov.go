package main

import (
	"bytes"
	"fmt"
	"github.com/kballard/goirc/irc"
	markov "github.com/mischief/bananaphone"
	"io/ioutil"
	"log"
	"strconv"
	"strings"
)

func init() {
	RegisterModule("markov", func() Module {
		return &MarkovMod{}
	})
}

type MarkovMod struct {
	chain *markov.Chain
}

func (m *MarkovMod) Init(b *Bot, conn irc.SafeConn) error {
	conf := b.Config.Search("mod", "markov")
	corpus := conf.Search("corpus")
	orders := conf.Search("order")
	nwords := conf.Search("nword")

	nword, err := strconv.Atoi(nwords)
	if err != nil {
		return fmt.Errorf("markov config: invalid nword %s: %s", nwords, err)
	}
	if nword < 1 || nword > 60 {
		return fmt.Errorf("markov config: nword out of range")
	}

	order, err := strconv.Atoi(orders)
	if err != nil {
		return fmt.Errorf("markov config: invalid order %s: %s", orders, err)
	}
	if order < 0 || order > 10 {
		return fmt.Errorf("markov config: order out of range")
	}

	m.chain = markov.NewChain(order)

	if corpus != "" {
		b, e := ioutil.ReadFile(corpus)
		if e == nil {
			m.chain.Build(bytes.NewReader(b))
		}
	}

	b.Hook("markov", func(b *Bot, sender, cmd string, args ...string) error {
		b.Conn.Privmsg(sender, m.chain.Generate(nword))
		return nil
	})

	conn.AddHandler("PRIVMSG", func(c *irc.Conn, l irc.Line) {
		if strings.HasPrefix(l.Args[0], b.Magic) {
			return
		}
		m.chain.Build(strings.NewReader(l.Args[1]))
	})

	log.Printf("markov module initialized with order %d nword %d corpus %s", order, nword, corpus)
	return nil
}

func (m *MarkovMod) Reload() error {
	return nil
}

func (m *MarkovMod) Call(args ...string) error {
	return nil
}
