package main

import (
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
	xkbLogger := logrus.WithField("module", "xkb")
	eventChan := make(chan x.GenericEvent)
	conn.AddEventChan(eventChan)

	for {
		event, ok := <-eventChan
		if !ok {
			xkbLogger.Error("xkb channel error")
			return
		}

		stateEvent, err := xkb.NewStateNotifyEvent(event)
		if err != nil {
			xkbLogger.Error(err)
		}
		xkbLogger.WithFields(logrus.Fields{"id": currentWindow, "group": stateEvent.LockedGroup}).Info("layout changed")
		layouts[currentWindow] = stateEvent.LockedGroup
	}
}

func main() {
	i3Logger := logrus.WithField("module", "i3")
	xkbLogger := logrus.WithField("module", "xkb")

	layouts = make(map[i3.NodeID]uint8)
	i3ER := i3.Subscribe(i3.WindowEventType, i3.ShutdownEventType)

	conn, err := x.NewConn()
	if err != nil {
		xkbLogger.Panic(err)
	}

	useExtCookie := xkb.UseExtension(conn, xkb.MajorVersion, xkb.MinorVersion)
	_, err = useExtCookie.Reply(conn)
	if err != nil {
		xkbLogger.Panic(err)
	}

	selectOpts := xkb.SelectDetail(xkb.EventTypeStateNotify, map[uint]bool{1 << 7: true})
	if err := xkb.SelectEventsChecked(conn, xkb.IDUseCoreKbd, selectOpts).Check(conn); err != nil {
		xkbLogger.Panic(err)
	}

	itree, err := i3.GetTree()
	if err != nil {
		i3Logger.Error(err)
		currentWindow = 0
	} else {
		currentWindow = itree.Root.FindFocused(func(n *i3.Node) bool { return n.Focused }).ID
	}

	go watchXkb(conn)

	for i3ER.Next() {
		switch i3ER.Event().(type) {
		case *i3.WindowEvent:
			event := i3ER.Event().(*i3.WindowEvent)
			switch event.Change {
			case "new":
				i3Logger.WithField("id", event.Container.ID).Info("new")
				layouts[event.Container.ID] = layouts[currentWindow]
			case "focus":
				i3Logger.WithField("id", event.Container.ID).Info("focus")
				currentWindow = event.Container.ID
				if err := xkb.LatchLockStateChecked(conn, xkb.IDUseCoreKbd, 0, 0, true, layouts[currentWindow], 0, 0, false, 0).Check(conn); err != nil {
					xkbLogger.WithField("id", currentWindow).Error(err)
				} else {
					xkbLogger.WithFields(logrus.Fields{"id": currentWindow, "group": layouts[currentWindow]}).Info("layout changed")
				}
			case "close":
				i3Logger.WithField("id", event.Container.ID).Info("close")
				delete(layouts, event.Container.ID)
			}
		case *i3.ShutdownEvent:
			i3Logger.Info("session shutdown")
			return
		}
	}
}
