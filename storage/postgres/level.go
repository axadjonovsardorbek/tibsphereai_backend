package postgres

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	m "tibsphereai/pkg/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// LevelRepo - statik "levels" jadvali, admin panel orqali CRUD qilinadi.
type LevelRepo struct {
	db *pgxpool.Pool
}

func NewLevelRepo(db *pgxpool.Pool) *LevelRepo {
	return &LevelRepo{db: db}
}

func (r *LevelRepo) Create(req *m.LevelCreateReq) (*m.ById, error) {
	id := uuid.NewString()
	_, err := r.db.Exec(context.Background(), `
		INSERT INTO levels (id, level_number, slug, required_stars, title, badge_image_url, stars_reward, coin_reward)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, id, req.LevelNumber, req.Slug, req.RequiredStars, req.Title, req.BadgeImageUrl, req.StarsReward, req.CoinReward)
	if err != nil {
		return nil, fmt.Errorf("error while creating level: %w", err)
	}
	return &m.ById{Id: id}, nil
}

func (r *LevelRepo) GetAll() ([]m.LevelRes, error) {
	rows, err := r.db.Query(context.Background(), `
		SELECT id, level_number, slug, required_stars, title, badge_image_url, stars_reward, coin_reward
		FROM levels ORDER BY required_stars ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("error while getting levels: %w", err)
	}
	defer rows.Close()

	var list []m.LevelRes
	for rows.Next() {
		l := m.LevelRes{}
		if err := rows.Scan(&l.Id, &l.LevelNumber, &l.Slug, &l.RequiredStars, &l.Title, &l.BadgeImageUrl, &l.StarsReward, &l.CoinReward); err != nil {
			return nil, fmt.Errorf("error while scanning level: %w", err)
		}
		list = append(list, l)
	}
	return list, nil
}

func (r *LevelRepo) Update(req *m.LevelUpdateReq) (*m.Void, error) {
	var conditions []string
	var args []interface{}

	conditions = append(conditions, "level_number = $"+strconv.Itoa(len(args)+1))
	args = append(args, req.LevelNumber)
	conditions = append(conditions, "required_stars = $"+strconv.Itoa(len(args)+1))
	args = append(args, req.RequiredStars)
	conditions = append(conditions, "stars_reward = $"+strconv.Itoa(len(args)+1))
	args = append(args, req.StarsReward)
	conditions = append(conditions, "coin_reward = $"+strconv.Itoa(len(args)+1))
	args = append(args, req.CoinReward)
	if req.Slug != "" && req.Slug != "string" {
		conditions = append(conditions, "slug = $"+strconv.Itoa(len(args)+1))
		args = append(args, req.Slug)
	}
	if req.Title != "" && req.Title != "string" {
		conditions = append(conditions, "title = $"+strconv.Itoa(len(args)+1))
		args = append(args, req.Title)
	}
	if req.BadgeImageUrl != "" && req.BadgeImageUrl != "string" {
		conditions = append(conditions, "badge_image_url = $"+strconv.Itoa(len(args)+1))
		args = append(args, req.BadgeImageUrl)
	}

	query := "UPDATE levels SET " + strings.Join(conditions, ", ")
	query += " WHERE id = $" + strconv.Itoa(len(args)+1)
	args = append(args, req.Id)

	res, err := r.db.Exec(context.Background(), query, args...)
	if err != nil {
		return nil, fmt.Errorf("error while updating level: %w", err)
	}
	if res.RowsAffected() == 0 {
		return nil, fmt.Errorf("level with id %s couldnt update", req.Id)
	}
	return &m.Void{}, nil
}

func (r *LevelRepo) Delete(req *m.ById) (*m.Void, error) {
	res, err := r.db.Exec(context.Background(), `DELETE FROM levels WHERE id = $1`, req.Id)
	if err != nil {
		return nil, fmt.Errorf("error while deleting level: %w", err)
	}
	if res.RowsAffected() == 0 {
		return nil, fmt.Errorf("level with id %s couldnt delete", req.Id)
	}
	return &m.Void{}, nil
}

func (r *LevelRepo) GetLevelForStars(stars int) (*m.LevelRes, error) {
	row := r.db.QueryRow(context.Background(), `
		SELECT id, level_number, slug, required_stars, title, badge_image_url, stars_reward, coin_reward
		FROM levels WHERE required_stars <= $1
		ORDER BY required_stars DESC LIMIT 1
	`, stars)

	l := m.LevelRes{}
	err := row.Scan(&l.Id, &l.LevelNumber, &l.Slug, &l.RequiredStars, &l.Title, &l.BadgeImageUrl, &l.StarsReward, &l.CoinReward)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("error while getting level for stars: %w", err)
	}
	return &l, nil
}
