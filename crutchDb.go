package main

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"regexp"
	"strconv"
	"strings"

	"github.com/gosimple/slug"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/log/logrusadapter"
	"github.com/jackc/pgx/v4/pgxpool"
)

type CrutchDBHelper struct {
	pool *pgxpool.Pool
}

type ApiCredentials struct {
	Enabled  bool   `json:"enabled"`
	Login    string `json:"login"`
	Password string `json:"password"`
	AuthType string `json:"authType"`
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

func generatePassword(passwordLength, minSpecialChar, minNum, minUpperCase int) string {
	var (
		lowerCharSet   = "abcdedfghijklmnopqrst"
		upperCharSet   = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
		specialCharSet = "!@#$%&*"
		numberSet      = "0123456789"
		allCharSet     = lowerCharSet + upperCharSet + specialCharSet + numberSet
	)

	var password strings.Builder

	//Set special character
	for i := 0; i < minSpecialChar; i++ {
		random := rand.Intn(len(specialCharSet))
		password.WriteString(string(specialCharSet[random]))
	}

	//Set numeric
	for i := 0; i < minNum; i++ {
		random := rand.Intn(len(numberSet))
		password.WriteString(string(numberSet[random]))
	}

	//Set uppercase
	for i := 0; i < minUpperCase; i++ {
		random := rand.Intn(len(upperCharSet))
		password.WriteString(string(upperCharSet[random]))
	}

	remainingLength := passwordLength - minSpecialChar - minNum - minUpperCase
	for i := 0; i < remainingLength; i++ {
		random := rand.Intn(len(allCharSet))
		password.WriteString(string(allCharSet[random]))
	}
	inRune := []rune(password.String())
	rand.Shuffle(len(inRune), func(i, j int) {
		inRune[i], inRune[j] = inRune[j], inRune[i]
	})
	return string(inRune)
}

func genApiLogin(userInfo UserInfo) string {

	re := regexp.MustCompile(`(.*)\.[^\.]+`)
	sl := re.ReplaceAllString(userInfo.Email, "$1")
	return slug.Make(sl)
}

func (db *CrutchDBHelper) getApiCredentials(userInfo UserInfo) (*ApiCredentials, error) {
	api := ApiCredentials{AuthType: "Basic"}

	login := genApiLogin(userInfo)
	suffix := ""

	for {
		err := db.pool.QueryRow(context.Background(), `
			WITH e AS(
				INSERT INTO api_credentials (user_id, login, enabled, password, date_created, date_updated) 
						 VALUES ($1, $2, False, $3, NOW(), NOW())
				ON CONFLICT(user_id) DO NOTHING
				RETURNING *
			)
			SELECT login, enabled, password FROM e
			UNION
			SELECT login, enabled, password FROM api_credentials WHERE user_id=$1
			`, userInfo.Id, login+suffix, generatePassword(16, 2, 2, 2)).Scan(&api.Login, &api.Enabled, &api.Password)

		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation /* or just == "23505" */ {
				suffix = strconv.Itoa(rand.Intn(100))
				continue
			}
			return nil, fmt.Errorf("Failed to retrieve user API credentials: %v", err)
		}
		break
	}

	return &api, nil

}

func (db *CrutchDBHelper) setApiCredentialsEnabled(userInfo UserInfo, enabled bool) (*ApiCredentials, error) {

	api := ApiCredentials{AuthType: "Basic"}
	err := db.pool.QueryRow(context.Background(), "UPDATE api_credentials SET enabled=$1, date_updated=NOW() WHERE user_id=$2 RETURNING login, enabled, password", enabled, userInfo.Id).Scan(&api.Login, &api.Enabled, &api.Password)

	return &api, err
}

func (db *CrutchDBHelper) updateApiCredentialsPassword(userInfo UserInfo) (*ApiCredentials, error) {

	api := ApiCredentials{AuthType: "Basic"}
	err := db.pool.QueryRow(context.Background(), "UPDATE api_credentials SET password=$1, date_updated=NOW() WHERE user_id=$2 RETURNING login, enabled, password", generatePassword(16, 2, 2, 2), userInfo.Id).Scan(&api.Login, &api.Enabled, &api.Password)

	return &api, err
}

func (db *CrutchDBHelper) getUserCredsFromApiLogin(login string) (int, string, error) {
	var password string
	var user_id int
	err := db.pool.QueryRow(context.Background(), "SELECT user_id, password FROM api_credentials WHERE login=$1 AND enabled=true", login).Scan(&user_id, &password)

	return user_id, password, err
}
