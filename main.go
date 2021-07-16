package main

import (
	"os"

	x "github.com/linuxdeepin/go-x11-client"
	xkb "github.com/linuxdeepin/go-x11-client/ext/xkb"
	"github.com/sirupsen/logrus"
	"go.i3wm.org/i3"
)

var (
	layouts       map[i3.NodeID]uint8
	currentWindow i3.NodeID
)

func watchXkb(conn *x.Conn) {
	eventChan := make(chan x.GenericEvent)
	conn.AddEventChan(eventChan)

	for {
		event, ok := <-eventChan
		if !ok {
			logrus.Error("xkb channel error")
			return
		}

		stateEvent, err := xkb.NewStateNotifyEvent(event)
		if err != nil {
			logrus.Error(err)
		}
		layouts[currentWindow] = stateEvent.LockedGroup
	}
}

func main() {
	logFile, err := os.OpenFile("/var/log/i3xkb", os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		logrus.Error(err)
	} else {
		logrus.SetOutput(logFile)
	}

	layouts = make(map[i3.NodeID]uint8)
	i3ER := i3.Subscribe(i3.WindowEventType)

	conn, err := x.NewConn()
	if err != nil {
		logrus.Panic(err)
	}

	useExtCookie := xkb.UseExtension(conn, xkb.MajorVersion, xkb.MinorVersion)
	_, err = useExtCookie.Reply(conn)
	if err != nil {
		logrus.Panic(err)
	}

	selectOpts := xkb.SelectDetail(xkb.EventTypeStateNotify, map[uint]bool{1 << 7: true})
	if err := xkb.SelectEventsChecked(conn, xkb.IDUseCoreKbd, selectOpts).Check(conn); err != nil {
		logrus.Panic(err)
	}

	itree, err := i3.GetTree()
	if err != nil {
		logrus.Error(err)
		currentWindow = 0
	} else {
		currentWindow = itree.Root.FindFocused(func(n *i3.Node) bool { return n.Focused }).ID
	}

	go watchXkb(conn)

	for i3ER.Next() {
		event := i3ER.Event().(*i3.WindowEvent)
		switch event.Change {
		case "new":
			layouts[event.Container.ID] = layouts[currentWindow]
		case "focus":
			currentWindow = event.Container.ID
			if err := xkb.LatchLockStateChecked(conn, xkb.IDUseCoreKbd, 0, 0, true, layouts[currentWindow], 0, 0, false, 0).Check(conn); err != nil {
				logrus.Error(err)
			}
		case "close":
			delete(layouts, event.Container.ID)
		}
	}
}
