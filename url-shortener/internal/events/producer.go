package events

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/segmentio/kafka-go"

	"github.com/AlexTihonow/url-shortener/internal/models"
)

type clickWriter interface {
	InsertClick(ctx context.Context, e models.ClickEvent) error
}

type Producer struct {
	writer   *kafka.Writer
	fallback clickWriter
	log      *slog.Logger
	ch       chan models.ClickEvent
	done     chan struct{}
}

func NewProducer(brokers, topic string, fallback clickWriter, log *slog.Logger) *Producer {
	p := &Producer{
		writer: &kafka.Writer{
			Addr:         kafka.TCP(brokers),
			Topic:        topic,
			Balancer:     &kafka.Hash{},
			BatchTimeout: 200 * time.Millisecond,
			RequiredAcks: kafka.RequireOne,
		},
		fallback: fallback,
		log:      log,
		ch:       make(chan models.ClickEvent, 1024),
		done:     make(chan struct{}),
	}
	go p.loop()
	return p
}

func (p *Producer) Publish(e models.ClickEvent) {
	select {
	case p.ch <- e:
	default:
		p.writeFallback(e)
	}
}

func (p *Producer) loop() {
	defer close(p.done)
	for e := range p.ch {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		payload, _ := json.Marshal(e)
		err := p.writer.WriteMessages(ctx, kafka.Message{
			Key:   []byte(e.ShortCode),
			Value: payload,
		})
		cancel()
		if err != nil {
			p.log.Warn("kafka publish failed, using db fallback", "err", err)
			p.writeFallback(e)
		}
	}
}

func (p *Producer) writeFallback(e models.ClickEvent) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := p.fallback.InsertClick(ctx, e); err != nil {
		p.log.Error("click fallback write failed", "err", err)
	}
}

func (p *Producer) Close() error {
	close(p.ch)
	<-p.done
	return p.writer.Close()
}
