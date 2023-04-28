package gocord

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
	"github.com/gorilla/websocket"
)

type Client struct {
	Token    string
	handlers map[string]func(*MessageCreate)
}

type Message struct {
	ID        string `json:"id"`
	ChannelID string `json:"channel_id"`
	Content   string `json:"content"`
}

type MessageCreate struct {
	Message *Message `json:"message"`
}

func NewClient(token string) *Client {
	return &Client{
		Token:    token,
		handlers: make(map[string]func(*MessageCreate)),
	}
}

func (c *Client) AddCommand(name string, handler func(*MessageCreate)) {
	c.handlers[name] = handler
}

func (c *Client) Start() error {
	u, _ := url.Parse("wss://gateway.discord.gg")
	q := u.Query()
	q.Set("v", "9")
	q.Set("encoding", "json")
	u.RawQuery = q.Encode()

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return err
	}

	// Identify
	identify := map[string]interface{}{
		"op": 2,
		"d": map[string]interface{}{
			"token": c.Token,
			"intents": 32509,
			"properties": map[string]interface{}{
				"$os":      "linux",
				"$browser": "my_library",
				"$device":  "my_library",
			},
		},
	}

	if err := conn.WriteJSON(identify); err != nil {
		return err
	}

	// Heartbeat
	go func() {
		for {
			time.Sleep(40 * time.Second)
			if err := conn.WriteMessage(websocket.TextMessage, []byte(`{"op":1,"d":0}`)); err != nil {
				fmt.Println(err)
				return
			}
		}
	}()

	// Receive Messages
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			return err
		}

		var messageCreate *MessageCreate
		if err := json.Unmarshal(message, &messageCreate); err != nil {
			continue
		}

		// Check for Commands
		for name, handler := range c.handlers {
			if strings.HasPrefix(messageCreate.Message.Content, name) {
				handler(messageCreate)
			}
		}
	}
}

func (c *Client) SendMessage(channelID string, content string) error {
	u, _ := url.Parse(fmt.Sprintf("https://discord.com/api/channels/%s/messages", channelID))

	data := url.Values{}
	data.Set("content", content)

	req, err := http.NewRequest("POST", u.String(), strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", fmt.Sprintf("Bot %s", c.Token))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}
