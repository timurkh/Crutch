package main

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/log/logrusadapter"
	"github.com/jackc/pgx/v4/pgxpool"
)

type CrutchDBHelper struct {
	pool *pgxpool.Pool
}

type ApiCredentials struct {
	enabled  bool
	login    string
	password string
}

func initCrutchDBHelper(host string, user string, password string, database string) (*CrutchDBHelper, error) {
	url := "postgres://" + user + ":" + password + "@" + host + "/" + database
	conf, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, err
	}
	conf.ConnConfig.Logger = logrusadapter.NewLogger(log)
	conf.ConnConfig.LogLevel = pgx.LogLevelTrace

	pool, err := pgxpool.ConnectConfig(context.Background(), conf)
	if err != nil {
		return nil, err
	}

	db := CrutchDBHelper{pool}
	return &db, nil
}

func (db *ProdDBHelper) getApiCredentials(userInfo UserInfo) (*ApiCredentials, error) {
	api := ApiCredentials{}
	err := db.pool.QueryRow(context.Background(), `
		SELECT enabled, login, password 
		FROM  api_crendetials
		WHERE login =$1`, userInfo.CompanySlug).Scan(&api.enabled, &api.login, &api.password)
	if err != nil {
		return nil, fmt.Errorf("Failed to retrieve user info: %v", err)
	}

	return &api, nil

}
