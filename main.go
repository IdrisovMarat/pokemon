package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/IdrisovMarat/pokemon/internal/pokecache"
)

var baseUrl = "https://pokeapi.co/api/v2/location-area/"

var cache *pokecache.Cache
var pokedex *pokecache.Pokedex

func init() {
	// Инициализируем кэш с интервалом 1 минута
	cache = pokecache.NewCache(45 * time.Second)
	pokedex = pokecache.NewPokedex()

}

type pokemonJson struct {
	ID                     int    `json:"id"`
	Name                   string `json:"name"`
	BaseExperience         int    `json:"base_experience"`
	Height                 int    `json:"height"`
	IsDefault              bool   `json:"is_default"`
	Order                  int    `json:"order"`
	Weight                 int    `json:"weight"`
	Abilities              []any  `json:"abilities"`
	Forms                  []any  `json:"forms"`
	GameIndices            []any  `json:"game_indices"`
	HeldItems              []any  `json:"held_items"`
	LocationAreaEncounters string `json:"location_area_encounters"`
	Moves                  []any  `json:"moves"`
	Species                any    `json:"species"`
	Sprites                any    `json:"sprites"`
	Cries                  any    `json:"cries"`
	Stats                  []any  `json:"stats"`
	Types                  []any  `json:"types"`
	PastTypes              []any  `json:"past_types"`
	PastAbilities          []any  `json:"past_abilities"`
}

type locationAreaJson struct {
	EncounterMethodRates []any               `json:"encounter_method_rates"`
	GameIndex            int                 `json:"game_index"`
	ID                   int                 `json:"id"`
	Location             any                 `json:"location"`
	Name                 string              `json:"name"`
	Names                []any               `json:"names"`
	PokemonEncounters    []PokemonEncounters `json:"pokemon_encounters"`
}

type Pokemon struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type PokemonEncounters struct {
	Pokemon        Pokemon `json:"pokemon"`
	VersionDetails []any   `json:"version_details"`
}

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
	callback    func(*config, string) error
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

// Вспомогательная функция для обработки данных из кэша
func processCachedData(cachedData []byte, cfg *config) error {
	var location locationJson
	err := json.Unmarshal(cachedData, &location)
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

	// Увеличиваем offset для следующего вызова
	cfg.offset += cfg.limit

	return nil
}

func commandExit(cfg *config, s string) error {
	fmt.Println("Closing the Pokedex... Goodbye!")
	cache.Stop()
	os.Exit(0)
	if os.ErrClosed != nil {
		return os.ErrClosed
	}
	return nil
}

// CatchPokemon пытается поймать покемона с учетом его базового опыта
// baseExp - базовый опыт покемона (чем выше, тем сложнее поймать)
// возвращает true если покемон пойман, false если нет
func CatchPokemon(baseExp int) bool {

	// Инициализируем генератор случайных чисел
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	// Определяем базовую вероятность поимки (можно настроить)
	baseCatchRate := 0.7 // 70% базовая вероятность

	// Рассчитываем модификатор сложности на основе базового опыта
	// Чем выше опыт, тем сложнее поймать
	difficultyModifier := 1.0 - float64(baseExp)/1000.0

	// Ограничиваем модификатор разумными пределами
	if difficultyModifier < 0.1 {
		difficultyModifier = 0.1 // Минимальный шанс 10%
	}
	if difficultyModifier > 0.9 {
		difficultyModifier = 0.9 // Максимальный шанс 90%
	}

	// Финальная вероятность поимки
	catchProbability := baseCatchRate * difficultyModifier

	// Генерируем случайное число от 0.0 до 1.0
	randomValue := rng.Float64()

	// Проверяем, удалось ли поймать
	return randomValue <= catchProbability
}

func commandCatch(cfg *config, pokemon string) error {
	baseUrl = "https://pokeapi.co/api/v2/pokemon/"
	// Формируем URL с текущим offset и limit
	url := fmt.Sprintf("%s/%s/", baseUrl, pokemon)

	// Проверяем кэш
	// if cachedData, found := cache.Get(url); found {
	// 	fmt.Println("...USING CACHE DATA...")
	// 	return processCachedData(cachedData, cfg)
	// }

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

	var pokemonmain pokemonJson

	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(&pokemonmain)
	if err != nil {
		return err
	}

	// responseData, err := json.Marshal(locationArea)
	// if err != nil {
	// 	return err
	// }

	// // Сохраняем в кэш
	// cache.Add(url, responseData)
	// fmt.Println("...DATA CACHED...")

	// // Обновляем конфигурацию
	// cfg.next = location.Next
	// cfg.previous = location.Previous

	// pokedex := make(map[string]Pokemon)
	fmt.Printf("Throwing a Pokeball at %s...", pokemonmain.Name)
	experience := pokemonmain.BaseExperience

	if CatchPokemon(experience) {
		pokedex.Add(pokemonmain.Name)
		fmt.Printf("\n%s was caught!", pokemonmain.Name)
	} else {
		fmt.Printf("\n%s escaped!", pokemonmain.Name)
	}

	return nil
}

func commandExplore(cfg *config, loc string) error {
	// Формируем URL с текущим offset и limit
	url := fmt.Sprintf("%s/%s/", baseUrl, loc)

	// Проверяем кэш
	// if cachedData, found := cache.Get(url); found {
	// 	fmt.Println("...USING CACHE DATA...")
	// 	return processCachedData(cachedData, cfg)
	// }

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

	var locationArea locationAreaJson

	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(&locationArea)
	if err != nil {
		return err
	}

	// responseData, err := json.Marshal(locationArea)
	// if err != nil {
	// 	return err
	// }

	// // Сохраняем в кэш
	// cache.Add(url, responseData)
	// fmt.Println("...DATA CACHED...")

	// // Обновляем конфигурацию
	// cfg.next = location.Next
	// cfg.previous = location.Previous

	for _, k := range locationArea.PokemonEncounters {
		fmt.Println(k.Pokemon.Name)
	}

	return nil
}

func commandMap(cfg *config, s string) error {
	// Формируем URL с текущим offset и limit
	url := fmt.Sprintf("%s?offset=%d&limit=%d", baseUrl, cfg.offset, cfg.limit)

	// Проверяем кэш
	if cachedData, found := cache.Get(url); found {
		fmt.Println("...USING CACHE DATA...")
		return processCachedData(cachedData, cfg)
	}

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

	responseData, err := json.Marshal(location)
	if err != nil {
		return err
	}

	// Сохраняем в кэш
	cache.Add(url, responseData)
	fmt.Println("...DATA CACHED...")

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

func commandMapb(cfg *config, s string) error {
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

	// Проверяем кэш
	if cachedData, found := cache.Get(url); found {
		fmt.Println("...USING CACHE DATA...")
		return processCachedData(cachedData, cfg)
	}

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

func commandHelp(cfg *config, s string) error {
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
		"explore": {
			name:        "explore",
			description: "lists pokemons of the location area",
			callback:    commandExplore,
		},
		"catch": {
			name:        "catch",
			description: "trying to catch pokemon",
			callback:    commandCatch,
		},
	}

	scanner := bufio.NewScanner(os.Stdin)

	defer cache.Stop()

	for {
		fmt.Print("\nPokedex > ")

		scanner.Scan()
		// input := strings.TrimSpace(scanner.Text())
		input := strings.Fields(scanner.Text())

		if input == nil {
			continue
		}
		// fmt.Println(input[0])
		// fmt.Println(input[1])
		if len(input) == 1 {
			input = append(input, "")
		}

		inputCommand, ok := commands[input[0]]
		if !ok {
			fmt.Println("Unknown command")
			continue
		}

		err := inputCommand.callback(&pageConfig, input[1])
		if err != nil {
			fmt.Println("something goes wrong after callback func")
			continue
		}
	}

}
