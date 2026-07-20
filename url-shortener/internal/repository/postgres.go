package repository

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/AlexTihonow/url-shortener/internal/models"
)

var (
	ErrNotFound = errors.New("link not found")

	ErrCodeTaken = errors.New("short code already taken")
)

type Repo struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repo { return &Repo{pool: pool} }

func (r *Repo) NextID(ctx context.Context) (int64, error) {
	var id int64
	err := r.pool.QueryRow(ctx, `SELECT nextval('links_id_seq')`).Scan(&id)
	return id, err
}

func (r *Repo) Insert(ctx context.Context, l models.Link) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO links (id, short_code, original_url, expires_at)
		 VALUES ($1, $2, $3, $4)`,
		l.ID, l.ShortCode, l.OriginalURL, l.ExpiresAt)
	if isUniqueViolation(err) {
		return ErrCodeTaken
	}
	return err
}

func (r *Repo) GetByCode(ctx context.Context, code string) (models.Link, error) {
	var l models.Link
	err := r.pool.QueryRow(ctx,
		`SELECT id, short_code, original_url, created_at, expires_at
		 FROM links
		 WHERE short_code = $1 AND (expires_at IS NULL OR expires_at > now())`,
		code).Scan(&l.ID, &l.ShortCode, &l.OriginalURL, &l.CreatedAt, &l.ExpiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return l, ErrNotFound
	}
	return l, err
}

func (r *Repo) Delete(ctx context.Context, code string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM links WHERE short_code = $1`, code)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repo) InsertClick(ctx context.Context, e models.ClickEvent) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO clicks (link_id, clicked_at, user_agent, referer)
		 SELECT id, $2, $3, $4 FROM links WHERE short_code = $1`,
		e.ShortCode, e.ClickedAt, e.UserAgent, e.Referer)
	return err
}

func (r *Repo) Stats(ctx context.Context, code string) (models.Stats, error) {
	var s models.Stats
	err := r.pool.QueryRow(ctx,
		`SELECT l.short_code, l.original_url,
		        count(c.id) AS total, max(c.clicked_at) AS last_click
		 FROM links l
		 LEFT JOIN clicks c ON c.link_id = l.id
		 WHERE l.short_code = $1
		 GROUP BY l.id`,
		code).Scan(&s.ShortCode, &s.OriginalURL, &s.TotalClicks, &s.LastClick)
	if errors.Is(err, pgx.ErrNoRows) {
		return s, ErrNotFound
	}
	if err != nil {
		return s, err
	}

	rows, err := r.pool.Query(ctx,
		`SELECT to_char(d.day, 'YYYY-MM-DD'), coalesce(count(c.id), 0)
		 FROM generate_series(
		          (current_date - interval '6 days'), current_date, interval '1 day'
		      ) AS d(day)
		 LEFT JOIN clicks c
		        ON date_trunc('day', c.clicked_at) = d.day
		       AND c.link_id = (SELECT id FROM links WHERE short_code = $1)
		 GROUP BY d.day
		 ORDER BY d.day`,
		code)
	if err != nil {
		return s, err
	}
	defer rows.Close()
	for rows.Next() {
		var dc models.DayCount
		if err := rows.Scan(&dc.Day, &dc.Clicks); err != nil {
			return s, err
		}
		s.Last7Days = append(s.Last7Days, dc)
	}
	return s, rows.Err()
}

func (r *Repo) Ping(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return r.pool.Ping(ctx)
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
