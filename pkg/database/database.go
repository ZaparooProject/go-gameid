package database

import (
	"encoding/csv"
	"fmt"
	"os"
)

type Game struct {
	ID    string
	Title string
}

func Load() ([]Game, error) {
	file, err := os.Open("dbs/PSX.data.tsv")
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.Comma = '\t'
	reader.FieldsPerRecord = -1
	reader.LazyQuotes = true

	games := make([]Game, 0)

	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read header: %w", err)
	}

	if len(header) == 0 {
		return nil, fmt.Errorf("no fields found")
	} else if header[0] != "ID" {
		return nil, fmt.Errorf("missing ID field")
	} else if header[1] != "title" {
		return nil, fmt.Errorf("missing title field")
	}

	for {
		record, err := reader.Read()
		if err != nil {
			break
		}

		games = append(games, Game{
			ID:    record[0],
			Title: record[1],
		})
	}

	if len(games) == 0 {
		return nil, fmt.Errorf("no games found")
	}

	fmt.Println(games[0])

	return games, nil
}
