package bxmpp

import (
	"github.com/42wim/matterbridge/bridge/config"
	log "github.com/Sirupsen/logrus"
	"github.com/mattn/go-xmpp"

	"strings"
	"time"
)

type Bxmpp struct {
	xc      *xmpp.Client
	xmppMap map[string]string
	*config.Config
	Remote chan config.Message
}

type FancyLog struct {
	xmpp *log.Entry
}

type Message struct {
	Text     string
	Channel  string
	Username string
}

var flog FancyLog

func init() {
	flog.xmpp = log.WithFields(log.Fields{"module": "xmpp"})
}

func New(config *config.Config, c chan config.Message) *Bxmpp {
	b := &Bxmpp{}
	b.xmppMap = make(map[string]string)
	b.Config = config
	b.Remote = c
	return b
}

func (b *Bxmpp) Connect() error {
	var err error
	flog.xmpp.Info("Trying XMPP connection")
	b.xc, err = b.createXMPP()
	if err != nil {
		flog.xmpp.Debugf("%#v", err)
		return err
	}
	flog.xmpp.Info("Connection succeeded")
	b.setupChannels()
	go b.handleXmpp()
	return nil
}

func (b *Bxmpp) Name() string {
	return "xmpp"
}

func (b *Bxmpp) Send(msg config.Message) error {
	b.xc.Send(xmpp.Chat{Type: "groupchat", Remote: msg.Channel + "@" + b.Xmpp.Muc, Text: msg.Username + msg.Text})
	return nil
}

func (b *Bxmpp) createXMPP() (*xmpp.Client, error) {
	options := xmpp.Options{
		Host:     b.Config.Xmpp.Server,
		User:     b.Config.Xmpp.Jid,
		Password: b.Config.Xmpp.Password,
		NoTLS:    true,
		StartTLS: true,
		//StartTLS:      false,
		Debug:                        true,
		Session:                      true,
		Status:                       "",
		StatusMessage:                "",
		Resource:                     "",
		InsecureAllowUnencryptedAuth: false,
		//InsecureAllowUnencryptedAuth: true,
	}
	var err error
	b.xc, err = options.NewClient()
	return b.xc, err
}

func (b *Bxmpp) setupChannels() {
	for _, val := range b.Config.Channel {
		flog.xmpp.Infof("Joining %s as %s", val.Xmpp, b.Xmpp.Nick)
		b.xc.JoinMUCNoHistory(val.Xmpp+"@"+b.Xmpp.Muc, b.Xmpp.Nick)
	}
}

func (b *Bxmpp) xmppKeepAlive() {
	go func() {
		ticker := time.NewTicker(90 * time.Second)
		for {
			select {
			case <-ticker.C:
				b.xc.Send(xmpp.Chat{})
			}
		}
	}()
}

func (b *Bxmpp) handleXmpp() error {
	for {
		m, err := b.xc.Recv()
		if err != nil {
			return err
		}
		switch v := m.(type) {
		case xmpp.Chat:
			var channel, nick string
			if v.Type == "groupchat" {
				s := strings.Split(v.Remote, "@")
				if len(s) == 2 {
					channel = s[0]
				}
				s = strings.Split(s[1], "/")
				if len(s) == 2 {
					nick = s[1]
				}
				if nick != b.Xmpp.Nick {
					flog.xmpp.Infof("sending message to remote %s %s %s", nick, v.Text, channel)
					b.Remote <- config.Message{Username: nick, Text: v.Text, Channel: channel, Origin: "xmpp"}
				}
			}
		case xmpp.Presence:
			// do nothing
		}
	}
}
