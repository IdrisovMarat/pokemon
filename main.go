package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

const baseUrl = "https://pokeapi.co/api/v2/location-area/"

type locationJson struct {
	Count    int       `json:"count"`
	Next     *string   `json:"next"`
	Previous *string   `json:"previous"`
	Results  []Results `json:"results"`
}
type Results struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type cliCommand struct {
	name        string
	description string
	callback    func(*config) error
}

type config struct {
	next     *string
	previous *string
	offset   int
	limit    int
}

var pageConfig = config{
	next:     nil,
	previous: nil,
	offset:   0,
	limit:    20,
}

func commandExit(cfg *config) error {
	fmt.Println("Closing the Pokedex... Goodbye!")
	os.Exit(0)
	if os.ErrClosed != nil {
		return os.ErrClosed
	}
	return nil
}

func commandMap(cfg *config) error {
	// Формируем URL с текущим offset и limit
	url := fmt.Sprintf("%s?offset=%d&limit=%d", baseUrl, cfg.offset, cfg.limit)

	client := http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP error: %s", resp.Status)
	}

	var location locationJson

	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(&location)
	if err != nil {
		return err
	}
	// Обновляем конфигурацию
	cfg.next = location.Next
	cfg.previous = location.Previous

	for _, k := range location.Results {
		fmt.Println(k.Name)
	}

	// Увеличиваем offset для следующего вызова
	cfg.offset += cfg.limit

	return nil
}

func commandMapb(cfg *config) error {
	// Проверяем, можно ли идти назад
	if cfg.offset <= cfg.limit {
		fmt.Println("You're on the first page. Cannot go back.")
		return nil
	}

	// Уменьшаем offset
	cfg.offset -= cfg.limit * 2 // Уменьшаем на два шага, т.к. commandMap увеличивает offset
	if cfg.offset < 0 {
		cfg.offset = 0
	}

	// Формируем URL
	url := fmt.Sprintf("%s?offset=%d&limit=%d", baseUrl, cfg.offset, cfg.limit)

	client := http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP error: %s", resp.Status)
	}

	var location locationJson
	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(&location)
	if err != nil {
		return err
	}

	// Обновляем конфигурацию
	cfg.next = location.Next
	cfg.previous = location.Previous

	// Выводим результаты
	for _, k := range location.Results {
		fmt.Println(k.Name)
	}

	// Увеличиваем offset для следующего вызова (если будут вызывать map)
	cfg.offset += cfg.limit

	return nil
}

func commandHelp(cfg *config) error {
	fmt.Println("Welcome to the Pokedex!")
	fmt.Println("Usage:")
	fmt.Println()
	fmt.Println("help: Displays a help message")
	fmt.Println("exit: Exit the Pokedex")
	fmt.Println("map: Display next 20 location areas")
	fmt.Println("mapb: Display previous 20 location areas")
	fmt.Println()

	return nil
}

func main() {

	var commands = map[string]cliCommand{
		"exit": {
			name:        "exit",
			description: "Exit the Pokedex",
			callback:    commandExit,
		},
		"help": {
			name:        "help",
			description: "description",
			callback:    commandHelp,
		},
		"map": {
			name:        "map",
			description: "lists next 20 Api resources",
			callback:    commandMap,
		},
		"mapb": {
			name:        "mapb",
			description: "lists previous 20 Api resources",
			callback:    commandMapb,
		},
	}

	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print("Pokedex > ")

		scanner.Scan()
		input := strings.TrimSpace(scanner.Text())

		if input == "" {
			continue
		}

		inputCommand, ok := commands[input]
		if !ok {
			fmt.Println("Unknown command")
			continue
		}

		err := inputCommand.callback(&pageConfig)
		if err != nil {
			fmt.Println("something goes wrong after callback func")
			continue
		}
	}

}

