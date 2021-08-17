package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
)

// try to open special URL with sessionid from cookies  if user is not authenticated or not authorized,
// he will be redirected to login page (if not authorized) or orders page (if user is a seller)
func checkAuth(r *http.Request) error {
	sessionCookie, err := r.Cookie("sessionid")

	if err != nil {
		err = fmt.Errorf("Не удалось получить ключ сессии: %w", err)
		return err
	}

	log.Println("Checking session " + sessionCookie.Value)
	checkBuyerURL := "https://industrial.market/accounts/portal_contractors/"

	req, err := http.NewRequest("GET", checkBuyerURL, nil)

	if err != nil {
		err = fmt.Errorf("Не удалось проверить сессию: %w", err)
		return err
	}

	req.AddCookie(sessionCookie)

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return errors.New("Поиск доступен только аутентифицированным клиентам")
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Failed to check current session: %w", err.Error())
		return errors.New("Текущий пользователь не авторизован для поиска по продуктовому каталогу")
	}

	finalURL := resp.Request.URL.String()
	if finalURL != checkBuyerURL {
		log.Printf("The URL you ended up at is: %v\n", finalURL)
		return errors.New("Поиск доступен только аутентифицированным клиентам")
	}

	defer resp.Body.Close()
	log.Printf("StatusCode: %d\n", resp.StatusCode)

	if resp.StatusCode != 200 {
		err = fmt.Errorf("Не удалось проверить аутентификацию, сервер вернул %", resp.StatusCode)
		return err
	}

	return nil
}
