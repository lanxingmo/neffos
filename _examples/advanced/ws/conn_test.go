package ws_test

import (
	"bytes"
	"sync"
	"testing"

	"github.com/kataras/fastws/_examples/advanced/ws"
)

func TestAsk(t *testing.T) {
	var (
		namespace   = "default"
		pingEvent   = "ping"
		pongMessage = []byte("PONG MESSAGE")
	)

	testMessage := func(dialer string, i int, msg ws.Message) {
		if msg.Namespace != namespace {
			t.Fatalf("[%s] [%d] expected namespace to be %s but got %s instead", dialer, i, namespace, msg.Namespace)
		}

		if msg.Event != pingEvent {
			t.Fatalf("[%s] [%d] expected event to be %s but got %s instead", dialer, i, pingEvent, msg.Event)
		}

		if !bytes.Equal(msg.Body, pongMessage) {
			t.Fatalf("[%s] [%d] from callback: expected %s but got %s", dialer, i, string(pongMessage), string(msg.Body))
		}
	}

	teardownServer := runTestServer("localhost:8080", ws.Namespaces{namespace: ws.Events{
		pingEvent: func(c ws.NSConn, msg ws.Message) error {
			// c.Emit("event", pongMessage)
			return ws.Reply(pongMessage) // changes only body; ns,event remains.
		}}})
	defer teardownServer()

	err := runTestClient("localhost:8080", ws.Namespaces{namespace: ws.Events{}}, func(dialer string, client *ws.Client) {
		defer client.Close()

		c, err := client.Connect(nil, namespace)
		if err != nil {
			t.Fatal(err)
		}

		for i := 1; i <= 5; i++ {
			msg := c.Ask(nil, pingEvent, nil)
			testMessage(dialer, i, msg)
		}

		msg := c.Ask(nil, pingEvent, nil)
		testMessage(dialer, -1, msg)
	})
	if err != nil {
		t.Fatal(err)
	}
}
func TestOnAnyEvent(t *testing.T) {
	var (
		namespace       = "default"
		expectedMessage = ws.Message{
			Namespace: namespace,
			Event:     "an_event",
			Body:      []byte("a_body"),
		}
		wg          sync.WaitGroup // a pure check for client's `Emit` to fire (`Ask` don't need this).
		testMessage = func(msg ws.Message) {
			// if !reflect.DeepEqual(msg, expectedMessage) { no becasue of Ask.wait.
			if msg.Namespace != expectedMessage.Namespace ||
				msg.Event != expectedMessage.Event ||
				!bytes.Equal(msg.Body, expectedMessage.Body) {

				t.Fatalf("expected message to be:\n%#+v\n\tbut got:\n%#+v", expectedMessage, msg)
			}
		}
	)

	teardownServer := runTestServer("localhost:8080", ws.Namespaces{namespace: ws.Events{
		ws.OnAnyEvent: func(c ws.NSConn, msg ws.Message) error {
			if ws.IsSystemEvent(msg.Event) { // skip connect/disconnect messages.
				return nil
			}

			return ws.Reply(msg.Body)
		}}})
	defer teardownServer()

	err := runTestClient("localhost:8080", ws.Namespaces{namespace: ws.Events{
		expectedMessage.Event: func(c ws.NSConn, msg ws.Message) error {
			defer wg.Done()
			testMessage(msg)

			return nil
		},
	}}, func(dialer string, client *ws.Client) {
		defer client.Close()

		c, err := client.Connect(nil, namespace)
		if err != nil {
			t.Fatal(err)
		}

		wg.Add(1)
		c.Emit(expectedMessage.Event, expectedMessage.Body)
		wg.Wait()

		testMessage(c.Ask(nil, expectedMessage.Event, expectedMessage.Body))
	})
	if err != nil {
		t.Fatal(err)
	}
}
