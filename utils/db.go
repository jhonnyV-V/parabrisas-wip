package utils

import (
	"database/sql"
	"errors"

	"github.com/jmoiron/sqlx"
	"github.com/mattn/go-sqlite3"
)

type WindshieldType string

// L = Left R = Right B = Back F = Front
const (
	LFDOOR     WindshieldType = "LFDOOR"
	RFDOOR     WindshieldType = "RFDOOR"
	LBDOOR     WindshieldType = "LBDOOR"
	RBDOOR     WindshieldType = "RBDOOR"
	WINDSHIELD WindshieldType = "WINDSHIELD"
	LFVENT     WindshieldType = "LFVENT"
	RFVENT     WindshieldType = "RFVENT"
	LBVENT     WindshieldType = "LBVENT"
	RBVENT     WindshieldType = "RBVENT"
	LBQUARTER  WindshieldType = "LBQUARTER"
	RBQUARTER  WindshieldType = "RBQUARTER"
	BACK       WindshieldType = "BACK"
)

type Brand struct {
	Id   int64  `db:"id"`
	Name string `db:"name"`
}

type Model struct {
	Id    int64  `db:"id"`
	Name  string `db:"name"`
	Brand int64  `db:"brand_id"`
}

type WindShield struct {
	Type  WindshieldType `db:"type"`
	Brand int64          `db:"brand_id"`
	Model int64          `db:"model_id"`
	Stock int            `db:"stock"`
	Year  string         `db:"year"`
	//should the year be here?
}

var (
	ErrDuplicate    = errors.New("record already exists")
	ErrNotExists    = errors.New("row not exists")
	ErrUpdateFailed = errors.New("update failed")
	ErrDeleteFailed = errors.New("delete failed")
)

type SQLiteRepository struct {
	db *sqlx.DB
}

var DB *SQLiteRepository

func OpenDB(driver string, datasource string) (*SQLiteRepository, error) {
	rawdb, err := sqlx.Connect(driver, datasource)
	if err != nil {
		return nil, err
	}
	DB = &SQLiteRepository{db: rawdb}
	return DB, nil
}

func (db *SQLiteRepository) Migrate() {
	setPragma := "PRAGMA foreign_keys;"

	createBrandTable := `
    CREATE TABLE IF NOT EXISTS brand(
      id         INTEGER PRIMARY KEY NOT NULL, 
      name			 TEXT UNIQUE NOT NULL,
    );
  `
	createModelTable := `
    CREATE TABLE IF NOT EXISTS model(
      id				INTEGER  PRIMARY KEY NOT NULL,
      name			TEXT UNIQUE NOT NULL,
			brand_id	INTEGER  NOT NULL,
			UNIQUE(id, brand_id),
      FOREIGN KEY(brand_id) REFERENCES brand(id) ON DELETE CASCADE,
    );
  `

	createWindshieldTable := `
    CREATE TABLE IF NOT EXISTS windshield(
      id				INTEGER  PRIMARY KEY NOT NULL,
      type			TEXT NOT NULL,
			year      TEXT NOT NULL,
      stock     INTEGER NOT NULL,
			brand_id  INTEGER NOT NULL,
      model_id  INTEGER NOT NULL,
      UNIQUE(id, model_id),
      FOREIGN KEY(model_id) REFERENCES model(id) ON DELETE CASCADE,
      FOREIGN KEY(brand_id) REFERENCES brand(id) ON DELETE CASCADE
    );
  `

	tx := db.db.MustBegin()

	tx.MustExec(setPragma)
	Logger.Info("setting pragma")

	tx.MustExec(createBrandTable)
	Logger.Info("setting user table")

	tx.MustExec(createModelTable)
	Logger.Info("setting content table")

	tx.MustExec(createWindshieldTable)
	Logger.Info("setting user_content table")

	//tx.MustExec(insertAdminAccount)
	//Logger.Info("inserting admin account");

	err := tx.Commit()
	if err != nil {
		Logger.Error(err.Error())
		panic(err)
	}
}

func (db *SQLiteRepository) CreateBrand(name string) (int64, error) {
	res, err := db.db.Exec("INSERT INTO brand(name) values(?)", name)
	if err != nil {
		var sqliteErr sqlite3.Error
		if errors.As(err, &sqliteErr) {
			if errors.Is(sqliteErr.ExtendedCode, sqlite3.ErrConstraintUnique) {
				return 0, ErrDuplicate
			}
		}
		return 0, err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}

	return id, nil
}

func (db *SQLiteRepository) CreateModel(name string, brand_id int64) (int64, error) {
	res, err := db.db.Exec("INSERT INTO model(name, brand_id) values(?, ?)", name, brand_id)
	if err != nil {
		var sqliteErr sqlite3.Error
		if errors.As(err, &sqliteErr) {
			if errors.Is(sqliteErr.ExtendedCode, sqlite3.ErrConstraintUnique) {
				return 0, ErrDuplicate
			}
		}
		return 0, err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}

	return id, nil
}

func (db *SQLiteRepository) CreateWindshield(typename WindshieldType, year string, stock, brand_id, model_id int64) (int64, error) {
	res, err := db.db.Exec(
		"INSERT INTO windshield(type, year, stock, brand_id, model_id) values(?, ?, ?, ?, ?)",
		typename, year, stock, brand_id, model_id,
	)
	if err != nil {
		var sqliteErr sqlite3.Error
		if errors.As(err, &sqliteErr) {
			if errors.Is(sqliteErr.ExtendedCode, sqlite3.ErrConstraintUnique) {
				return 0, ErrDuplicate
			}
		}
		return 0, err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}

	return id, nil
}

func (db *SQLiteRepository) UpdateWindshieldStock(id, stock int64) error {
	_, err := db.db.Exec("UPDATE windshield SET stock=? WHERE id=?", stock, id)
	if err != nil {
		return err
	}

	return nil
}

func (db *SQLiteRepository) GetAllBrands() ([]Brand, error) {
	var brands []Brand
	err := db.db.Get(&brands, "SELECT * FROM brand")

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return []Brand{}, nil
		}
		return []Brand{}, err
	}

	return brands, nil
}

func (db *SQLiteRepository) GetModelsByBrandId(id int64) ([]Model, error) {
	var models []Model
	err := db.db.Get(&models, "SELECT * FROM model WHERE brand_id=?", id)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return []Model{}, nil
		}
		return []Model{}, err
	}

	return models, nil
}

func (db *SQLiteRepository) GetModelsByBrandName(name string) ([]Model, error) {
	var models []Model
	err := db.db.Get(
		&models,
		`SELECT
			m.name,
			m.id,
			m.brand_id,
			b.name AS brand_name
		FROM model m
			JOIN brand b ON m.brand_id = b.id
			WHERE b.name = ?`,
		name,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return []Model{}, nil
		}
		return []Model{}, err
	}

	return models, nil
}

func (db *SQLiteRepository) GetWindShieldsByModelId(id int64) ([]WindShield, error) {
	var windshields []WindShield
	err := db.db.Get(&windshields, "SELECT * FROM windShield WHERE model_id = ?", id)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return []WindShield{}, nil
		}
		return []WindShield{}, err
	}

	return windshields, nil
}
