package events

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"

	"github.com/segmentio/kafka-go"

	"github.com/AlexTihonow/url-shortener/internal/models"
)

type Consumer struct {
	reader *kafka.Reader
	sink   clickWriter
	log    *slog.Logger
}

func NewConsumer(brokers, topic string, sink clickWriter, log *slog.Logger) *Consumer {
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{brokers},
		Topic:   topic,
		GroupID: "click-persister",
	})
	return &Consumer{reader: r, sink: sink, log: log}
}

func (c *Consumer) Run(ctx context.Context) {
	for {
		msg, err := c.reader.ReadMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return
			}
			c.log.Warn("kafka read error", "err", err)
			continue
		}
		var e models.ClickEvent
		if err := json.Unmarshal(msg.Value, &e); err != nil {
			c.log.Warn("bad click payload, skipping", "err", err)
			continue
		}
		if err := c.sink.InsertClick(ctx, e); err != nil {
			c.log.Error("persist click failed", "err", err)
		}
	}
}

func (c *Consumer) Close() error { return c.reader.Close() }
