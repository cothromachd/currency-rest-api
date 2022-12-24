package main

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"

	//"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// currency exchange api-key: 41f50f0fae6e5261c7bcd76e374483be
type CurExchange struct {
	Currency_from string    `json:"currency_from,omitempty"`
	Currency_to   string    `json:"currency_to,omitempty"`
	Well          float64   `json:"well,omitempty"`
	Updated_at    time.Time `json:"updated_at,omitempty"`
}

type pairsForUpdater1 struct {
	Currency_from string `json:"currency_from,omitempty"`
	Currency_to   string `json:"currency_to,omitempty"`
}

type pairsForUpdater2 struct {
	Well       float64   `json:"well,omitempty"`
	Updated_at time.Time `json:"updated_at,omitempty"`
}

func main() {
	app := fiber.New()
	//ежечасный обновлятор для наших курсов валют из базы данных
	//начинается

	//заканчивается
	dbPool, err := pgxpool.New(context.Background(), "postgres://khalidmagnificent:190204@localhost:5432/currency")
	if err != nil {

		log.Printf("%+v", err)
		fiber.NewError(fiber.StatusServiceUnavailable, err.Error())

	}

	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	// updater
	go func() {

		for range ticker.C {
			var currenciesNames []*pairsForUpdater1
			var currenciesRateAndTime pairsForUpdater2

			rows, err := dbPool.Query(context.Background(), `SELECT currency_from, currency_to FROM currency_rates;`)
			if err != nil {
				log.Panic(err)
			}

			for rows.Next() {

				sp := &pairsForUpdater1{}

				rows.Scan(&sp.Currency_from, &sp.Currency_to)

				currenciesNames = append(currenciesNames, sp)

				if err = rows.Err(); err != nil {
					log.Panic(err)
				}

			}

			for _, elem := range currenciesNames {

				url := "https://currate.ru/api/?get=rates&pairs=" + elem.Currency_from + elem.Currency_to + "&key=41f50f0fae6e5261c7bcd76e374483be"

				resp, err := http.Get(url)
				if err != nil {
					log.Panic(err)
				}

				currenciesRateAndTime.Updated_at, err = time.Parse("2006-01-02 15:04:05-07", time.Now().UTC().Format("2006-01-02 15:04:05-07"))
				if err != nil {
					log.Panic(err)
				}

				ratesJson, err := io.ReadAll(resp.Body)
				if err != nil {
					log.Panic(err)
				}

				ratesMap := make(map[string]interface{})

				err = json.Unmarshal(ratesJson, &ratesMap)
				if err != nil {
					log.Panic(err)
				}

				pairAndValueMapI := ratesMap["data"]

				pairAndValueMap := pairAndValueMapI.(map[string]interface{})

				pairAndValue := pairAndValueMap[elem.Currency_from+elem.Currency_to].(string)

				currenciesRateAndTime.Well, err = strconv.ParseFloat(pairAndValue, 64)
				if err != nil {
					log.Panic(err)
				}

				_, err = dbPool.Exec(context.Background(), `UPDATE currency_rates SET well = $1, updated_at = $2 WHERE currency_from = $3 AND currency_to = $4`, &currenciesRateAndTime.Well, &currenciesRateAndTime.Updated_at, elem.Currency_from, elem.Currency_to)
				if err != nil {

					log.Panic(err)

				}

			}
			log.Println("records updated")
		}

	}()

	//ручка для метода POST
	app.Post("/api/currency", func(c *fiber.Ctx) error {
		//объявляем объект модели нашей записи
		var s CurExchange

		//получаем json из http request body от клиента
		reqBody := c.Body()

		//десериализируем в объект
		json.Unmarshal(reqBody, &s)

		//составляем URL HTTP Get запроса
		url := "https://currate.ru/api/?get=rates&pairs=" + s.Currency_from + s.Currency_to + "&key=41f50f0fae6e5261c7bcd76e374483be"

		//запрашиваем методом GET на REST API курсов валют
		resp, err := http.Get(url)
		if err != nil {

			log.Printf("%+v", err)
			return fiber.NewError(fiber.StatusServiceUnavailable, err.Error())

		}

		//сохраняем время запроса
		s.Updated_at, err = time.Parse("2006-01-02 15:04:05-07", time.Now().UTC().Format("2006-01-02 15:04:05-07"))
		if err != nil {

			log.Printf("%+v", err)
			return fiber.NewError(fiber.StatusServiceUnavailable, err.Error())

		}

		//считываем из http тела полученного в ответ на GET запрос
		ratesJson, err := io.ReadAll(resp.Body)
		if err != nil {

			log.Printf("%+v", err)
			return fiber.NewError(fiber.StatusServiceUnavailable, err.Error())

		}

		//создаем мапу для десериализации json'а с курсами нужных нам валют
		ratesMap := make(map[string]interface{})

		err = json.Unmarshal(ratesJson, &ratesMap)
		if err != nil {
			log.Printf("%+v", err)
		}

		//мутка, чтобы достать мапу с курсами валют из мапы с http ответом
		//начинается
		pairAndValueMapI := ratesMap["data"]

		pairAndValueMap := pairAndValueMapI.(map[string]interface{})

		pairAndValue := pairAndValueMap[s.Currency_from+s.Currency_to].(string)
		//заканчивается

		//вставляем значение курса валют в поле нашего объекта, которое для этого предназначено
		s.Well, err = strconv.ParseFloat(pairAndValue, 64)
		if err != nil {
			log.Printf("%+v", err)
			return fiber.NewError(fiber.StatusServiceUnavailable, err.Error())
		}

		//создаем запись в нашей базе данных
		result, err := dbPool.Exec(context.Background(), `INSERT INTO currency_rates(currency_from, 
		currency_to, well, updated_at) VALUES ($1, $2, $3, $4);`, &s.Currency_from, &s.Currency_to, &s.Well, &s.Updated_at)
		if err != nil {

			log.Printf("%+v", err)
			c.Send([]byte(`{"status": "fail"`))
			return fiber.NewError(fiber.StatusServiceUnavailable, err.Error())

		}

		//проверяем на успешность, если да, отправляем 200 (успешно)
		if result.Insert() {
			log.Printf("%+v", err)
			c.Send([]byte(`{"status": "success"}`))
		}

		return nil

	})

	app.Put("/api/currency", func(c *fiber.Ctx) error {
		body := c.Body()
		bodyMap := make(map[string]interface{})
		json.Unmarshal(body, &bodyMap)
		time_of_update, err := time.Parse("2006-01-02 15:04:05-07", time.Now().UTC().Format("2006-01-02 15:04:05-07"))
		if err != nil {
			log.Printf("%+v", err)
			return fiber.NewError(fiber.StatusServiceUnavailable, err.Error())
		}

		_, err = dbPool.Exec(context.Background(), `UPDATE currency_rates SET well = $1, updated_at = $2 WHERE currency_from = $3 AND currency_to = $4;`,
			bodyMap["value"].(float64), time_of_update, bodyMap["currency_from"].(string), bodyMap["currency_to"].(string))
		if err != nil {
			log.Printf("%+v", err)
			return fiber.NewError(fiber.StatusServiceUnavailable, err.Error())
		}

		log.Println("PUT request is success")
		c.Send([]byte(`{"status":"success"}`))
		return nil
	})

	app.Get("/api/currency", func(c *fiber.Ctx) error {
		
		var s_arr []*CurExchange

		rows, err := dbPool.Query(context.Background(), `SELECT * FROM currency_rates;`)
		if err != nil {
			log.Printf("%+v", err)
			return fiber.NewError(fiber.StatusServiceUnavailable, err.Error())
		}

		for rows.Next() {
			s := &CurExchange{}

			err = rows.Scan(&s.Currency_from, &s.Currency_to, &s.Well, &s.Updated_at)
			if err != nil {
				log.Printf("%+v", err)
			}

			s_arr = append(s_arr, s)
		}
		s_arr_json, err := json.Marshal(s_arr)
		if err != nil {
			log.Printf("%+v", err)
		}
		c.Send(s_arr_json)
		return nil
	})
	log.Fatal(app.Listen(":4000"))

}
