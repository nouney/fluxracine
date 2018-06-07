package main

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/nouney/fluxracine/pkg/chat"
	"github.com/nouney/fluxracine/pkg/event"
	"github.com/pkg/errors"
)

type messagePayload struct {
	To      string `json:"to"`
	Message string `json:"message"`
}

func handleEventUserSendMessage(sess *chat.Session) event.Handler {
	return func(data interface{}) error {
		payload := &messagePayload{}
		err := json.Unmarshal(data.([]byte), &payload)
		if err != nil {
			return errors.Wrap(err, "json unmarshal")
		}

		log.Printf("user \"%s\" send \"%s\" to \"%s\"", sess.Nickname, payload.Message, payload.To)
		err = sess.SendMessage(payload.To, payload.Message)
		if err != nil {
			return errors.Wrap(err, "send message")
		}
		return nil
	}
}

type action struct {
	Action string      `json:"action"`
	Data   interface{} `json:"data"`
}

type receiveMessageData struct {
	From    string `json:"from"`
	Message string `json:"message"`
}

func handleEventUserReceiveMessage(sess *chat.Session, c *websocket.Conn) event.Handler {
	return func(data interface{}) error {
		log.Printf("user \"%s\" receive a message: %+v", sess.Nickname, data)
		msg := data.(*chat.MessagePayload)
		return c.WriteJSON(&action{
			Action: actionReceiveMessage,
			Data: &receiveMessageData{
				From:    msg.From,
				Message: msg.Message,
			},
		})
	}
}

func handleEventUserLogout(sess *chat.Session) event.Handler {
	return func(data interface{}) error {
		log.Printf("user \"%s\" disconnected", sess.Nickname)
		return sess.Close()
	}
}

// handleChatSession handles a chat session via a websocket.
func handleChatSession(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("upgrader:", err)
		return
	}
	defer c.Close()

	sess, err := server.NewSession()
	if err != nil {
		log.Println("new session:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	log.Printf("user \"%s\" logged in", sess.Nickname)
	log.Printf("nb sessions: %d", server.NbSessions())

	// Use the websocket and the chat server as event sources
	d := event.NewDispatcher(&websocketEventSource{c}, &chatSessionEventSource{sess})
	d.Handle(event.EventUserSendMessage, handleEventUserSendMessage(sess))
	d.Handle(event.EventUserReceiveMessage, handleEventUserReceiveMessage(sess, c))
	d.Handle(event.EventUserLogout, handleEventUserLogout(sess))

	err = d.Listen()
	if err != nil {
		log.Println("dispatcher:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func handleHome(w http.ResponseWriter, r *http.Request) {
	homeHTML.Execute(w, "ws://"+r.Host+"/chat")
}

var (
	homeHTML = template.Must(template.New("").Parse(`
<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<script>  
window.addEventListener("load", function(evt) {

	var output = document.getElementById("output");
	var input = document.getElementById("input");
	var receiver = document.getElementById("receiver");
	var ws;
	var print = function(message) {
		var d = document.createElement("div");
		d.innerHTML = message;
		output.appendChild(d);
	};

	var sendMessage = function(to, message) {
		var data = {
			"action": "send_message",
			"data": {
				"to": to,
				"message": message,
			}
		}
		ws.send(JSON.stringify(data));
		console.log("SEND:", data);
	};

	document.getElementById("open").onclick = function(evt) {
		if (ws) {
			return false;
		}
		ws = new WebSocket("{{.}}");
		ws.onopen = function(evt) {
			print("Connection established.");
		}
		ws.onclose = function(evt) {
			print("Disconnected.");
			ws = null;
		}
		ws.onmessage = function(evt) {
			console.log("RESPONSE:", evt.data);
			var msg = JSON.parse(evt.data);
			print("[FROM "+ msg.data.from + "] " + msg.data.message);
		}
		ws.onerror = function(evt) {
			print("ERROR: " + evt.data);
		}
		return false;
	};

	document.getElementById("send").onclick = function(evt) {
		if (!ws) {
			return false;
		}
		print("[TO " + receiver.value + "] " + input.value)
		sendMessage(receiver.value, input.value);
		return false;
	};
	
	document.getElementById("close").onclick = function(evt) {
		if (!ws) {
			return false;
		}
		ws.close();
		return false;
	};
});
</script>
</head>
<body>
<table>
	<tr>
		<td valign="top" width="50%">
			<p>
				Click "Open" to create a connection to the server, 
				"Send" to send a message to the server and "Close" to close the connection. 
				You can change the message and send multiple times.
			</p>
			<p>
				<form>
					<p>
						<button id="open">Open</button>
						<button id="close">Close</button>
					</p>
					<p>
						<input id="receiver" type="text" value="receiver">
						<input id="input" type="text" value="Hello world!">
						<button id="send">Send</button>
					</p>
				</form>
			</p>
		</td>
		<td valign="top" width="50%">
			<div id="output"></div>
		</td>
	</tr>
</table>
</body>
</html>`))
)
