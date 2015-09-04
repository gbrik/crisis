package crisis

import (
	"database/sql"
	"fmt"
	"log"
)

const (
	DB_USER     = "adminmx2q25f"
	DB_PASSWORD = "xjt2j3wAKmrZ"
	DB_NAME     = "crisismap"
)

type Database struct {
	db *sql.DB
}

var m_database *Database

func GetDatabaseInstance() *Database {
	if m_database == nil {
		dbinfo := fmt.Sprintf("user=%s password=%s dbname=%s sslmode=disable",
			DB_USER, DB_PASSWORD, DB_NAME)
		db, err := sql.Open("postgres", dbinfo)
		if err != nil {
			log.Fatal(err)
		}
		m_database = &Database{db}
	}
	return m_database
}

func (db *Database) Close() {
	db.db.Close()
}

func (db *Database) GetCrisisDivisions(crisis_id int) map[int][]*Division {
	rows, err := db.db.Query("SELECT faction.id, faction.faction_name, division.id, "+
		"division.coord_x, division.coord_y, division.division_name "+
		"FROM division INNER JOIN faction ON (faction.id = division.faction) "+
		"WHERE faction.crisis=?", crisis_id)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	return db.getCrisisDivisionsFromRows(rows)
}

func (db *Database) GetFactionDivisions(faction_id int) []*Division {
	rows, err := db.db.Query("SELECT faction.faction_name, division.id, division.coord_x, "+
		"division.coord_y, division.division_name "+
		"FROM division INNER JOIN faction ON (faction.id = division.faction) "+
		"INNER JOIN division_view ON (division_view.division_id = division.id) "+
		"WHERE division_view.faction_id=?", faction_id)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	return db.getFactionDivisionsFromRows(rows)
}

func (db *Database) getCrisisDivisionsFromRows(rows *sql.Rows) map[int][]*Division {
	m := make(map[int][]*Division)
	var facId int
	for rows.Next() {
		div := Division{}
		err := rows.Scan(&facId, &div.FacName, &div.Id, &div.CoordX, &div.CoordY, &div.DivName)
		if err != nil {
			log.Fatal(err)
		}
		db.loadUnitsFor(&div)
		m[facId] = append(m[facId], &div)
	}
	return m
}

func (db *Database) getFactionDivisionsFromRows(rows *sql.Rows) []*Division {
	var fs []*Division
	for rows.Next() {
		div := Division{}
		err := rows.Scan(&div.FacName, &div.Id, &div.CoordX, &div.CoordY, &div.DivName)
		if err != nil {
			log.Fatal(err)
		}
		db.loadUnitsFor(&div)
		fs = append(fs, &div)
	}
	return fs
}

func (db *Database) loadUnitsFor(div *Division) {
	rows, err := db.db.Query("SELECT unit_type.unit_name, unit.amount, unit_type.unit_speed "+
		"FROM unit INNER JOIN unit_type ON (unit.unit_type = unit_type.id)"+
		"WHERE unit.division = ?", div.Id)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	var (
		name   string
		amount int
		speed  int
	)
	min_speed := 1<<16 - 1
	for rows.Next() {
		if err = rows.Scan(&name, &amount, &speed); err != nil {
			log.Fatal(err)
		}
		div.Units[name] = amount
		if speed < min_speed {
			min_speed = speed
		}
	}
	div.Speed = min_speed
}