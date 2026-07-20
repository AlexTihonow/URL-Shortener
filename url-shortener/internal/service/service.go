package service

import (
	"context"
	"errors"
	"net/url"
	"time"

	"github.com/AlexTihonow/url-shortener/internal/models"
)

var ErrInvalidURL = errors.New("invalid url: must be absolute http(s)")

type repo interface {
	NextID(ctx context.Context) (int64, error)
	Insert(ctx context.Context, l models.Link) error
	GetByCode(ctx context.Context, code string) (models.Link, error)
	Delete(ctx context.Context, code string) error
	Stats(ctx context.Context, code string) (models.Stats, error)
}

type cacheLayer interface {
	Get(ctx context.Context, code string) (string, bool)
	Set(ctx context.Context, code, url string)
	Invalidate(ctx context.Context, code string)
}

type clickPublisher interface {
	Publish(e models.ClickEvent)
}

type Service struct {
	repo   repo
	cache  cacheLayer
	clicks clickPublisher
}

func New(r repo, c cacheLayer, p clickPublisher) *Service {
	return &Service{repo: r, cache: c, clicks: p}
}

type CreateInput struct {
	OriginalURL string
	CustomCode  string
	ExpiresAt   *time.Time
}

func (s *Service) Create(ctx context.Context, in CreateInput) (models.Link, error) {
	if !validURL(in.OriginalURL) {
		return models.Link{}, ErrInvalidURL
	}

	id, err := s.repo.NextID(ctx)
	if err != nil {
		return models.Link{}, err
	}

	code := in.CustomCode
	if code == "" {
		code = codeForID(id)
	}

	link := models.Link{
		ID:          id,
		ShortCode:   code,
		OriginalURL: in.OriginalURL,
		ExpiresAt:   in.ExpiresAt,
	}
	if err := s.repo.Insert(ctx, link); err != nil {
		return models.Link{}, err
	}
	link.CreatedAt = time.Now()
	return link, nil
}

func (s *Service) Resolve(ctx context.Context, code, userAgent, referer string) (string, error) {
	if target, ok := s.cache.Get(ctx, code); ok {
		s.trackClick(code, userAgent, referer)
		return target, nil
	}

	link, err := s.repo.GetByCode(ctx, code)
	if err != nil {
		return "", err
	}
	s.cache.Set(ctx, code, link.OriginalURL)
	s.trackClick(code, userAgent, referer)
	return link.OriginalURL, nil
}

func (s *Service) Delete(ctx context.Context, code string) error {
	if err := s.repo.Delete(ctx, code); err != nil {
		return err
	}
	s.cache.Invalidate(ctx, code)
	return nil
}

func (s *Service) Stats(ctx context.Context, code string) (models.Stats, error) {
	return s.repo.Stats(ctx, code)
}

func (s *Service) trackClick(code, userAgent, referer string) {
	s.clicks.Publish(models.ClickEvent{
		ShortCode: code,
		ClickedAt: time.Now().UTC(),
		UserAgent: userAgent,
		Referer:   referer,
	})
}

func validURL(raw string) bool {
	u, err := url.ParseRequestURI(raw)
	if err != nil {
		return false
	}
	return u.Scheme == "http" || u.Scheme == "https"
}
