package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/tristenkelly/pokedex/internal/pokecache"
)

type cliCommand struct {
	name        string
	description string
	callback    func(cfg *Config, args ...string) error
}

type Config struct {
	Next      *string
	Previous  *string
	cache     *pokecache.Cache
	area_name *string
	pokedex   map[string]Pokemon
}

type locationAreaResponse struct {
	Results  []LocationArea `json:"results"`
	Next     *string        `json:"next"`
	Previous *string        `json:"previous"`
}

type LocationArea struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type AreaEncounters struct {
	PokemonEncounters []PokemonEncounter `json:"pokemon_encounters"`
}

type PokemonEncounter struct {
	Pokemon NamedAPIResource `json:"pokemon"`
}

type NamedAPIResource struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type Pokemon struct {
	Name           string `json:"name"`
	Height         int    `json:"height"`
	BaseExperience int    `json:"base_experience"`
}

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	cfg := &Config{
		cache: pokecache.NewCache(30 * time.Second),
	}

	for {
		fmt.Print("PokeDex > ")
		userInput := ""
		if scanner.Scan() {
			userInput = scanner.Text()
		}
		parts := strings.Fields(userInput)
		if len(parts) == 0 {
			continue
		}

		name := strings.Join(parts[1:], " ")
		cfg.area_name = &name
		commandName := parts[0]
		args := parts[1:]

		if cmd, exists := commands[commandName]; exists {
			err := cmd.callback(cfg, args...)
			if err != nil {
				fmt.Println("Error:", err)
			}
		} else {
			fmt.Println("Unknown command")
		}

	}
}

func processURL[T any](url string, target *T) ([]byte, error) {
	res, err := http.Get(url)
	if err != nil {
		fmt.Println("Error fetching location areas:", err)
		return nil, err
	}

	defer res.Body.Close()

	data, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return nil, err
	}

	err = json.Unmarshal(data, target)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling JSON: %w", err)
	}
	return data, nil
}

func calculateChance(pokemon Pokemon) bool {
	maxChance := 100
	difficulty := pokemon.BaseExperience / 5

	if difficulty > maxChance {
		difficulty = maxChance
	}

	roll := rand.Intn(100)

	return roll > difficulty
}

func commandExit(cfg *Config, args ...string) error {
	fmt.Print("Closing the Pokedex... Goodbye!\n")
	os.Exit(0)
	return nil
}

func commandHelp(cfg *Config, args ...string) error {
	fmt.Println("\nWelcome to the Pokedex!")
	fmt.Println("Usage:")
	fmt.Println("help: displays this help message")
	fmt.Println("exit: exits the PokeDex")
	return nil
}

func commandMap(cfg *Config, args ...string) error {
	var url string

	if cfg.Next != nil && *cfg.Next != "" {
		url = *cfg.Next
	} else {
		fmt.Println("You're on the first page")
		url = "https://pokeapi.co/api/v2/location-area/?limit=20"
	}

	if _, ok := cfg.cache.Get(url); ok {
		fmt.Println("Using cached data for:", url)
		var locationData locationAreaResponse
		_, err := processURL(url, &locationData)
		if err != nil {
			return fmt.Errorf("error mapping pokemon: %w", err)
		}
		cfg.Next = locationData.Next
		for _, area := range locationData.Results {
			fmt.Printf("%v\n", area.Name)
		}
		return nil
	}
	var locationData locationAreaResponse

	body, err := processURL(url, &locationData)
	if err != nil {
		return err
	}

	cfg.cache.Add(url, body)
	cfg.Next = locationData.Next

	for _, area := range locationData.Results {
		fmt.Printf("%v\n", area.Name)
	}

	return nil
}

func commandMapb(cfg *Config, args ...string) error {
	var url string

	if cfg.Previous != nil && *cfg.Previous != "" {
		url = *cfg.Previous
	} else {
		url = "https://pokeapi.co/api/v2/location-area/?limit=20"
	}
	var locationData locationAreaResponse
	_, err := processURL(url, &locationData)
	if err != nil {
		return err
	}

	cfg.Previous = locationData.Previous

	for _, area := range locationData.Results {
		fmt.Printf("%v\n", area.Name)
	}
	return nil
}

func commandExplore(cfg *Config, args ...string) error {
	url := "https://pokeapi.co/api/v2/location-area/"
	fullURL := url + *cfg.area_name
	if data, ok := cfg.cache.Get(fullURL); ok {
		fmt.Println("Using cached data for:", fullURL)
		var pokemonData AreaEncounters
		if err := json.Unmarshal(data, &pokemonData); err != nil {
			fmt.Println("Error unmarshalling cached data:", err)
			return err
		}
		for _, encounter := range pokemonData.PokemonEncounters {
			fmt.Printf("%v\n", encounter.Pokemon.Name)
		}
		return nil
	}

	var pokemonData AreaEncounters

	body, err := processURL(fullURL, &pokemonData)
	if err != nil {
		return err
	}
	cfg.cache.Add(fullURL, body)

	fmt.Printf("Exploring %v\n", *cfg.area_name)
	fmt.Printf("Found Pokemon:\n")
	for _, encounter := range pokemonData.PokemonEncounters {
		fmt.Printf("- %v\n", encounter.Pokemon.Name)
	}

	return nil
}

func commandCatch(cfg *Config, args ...string) error {
	if len(args) != 1 {
		return fmt.Errorf("incorrect usage")
	}

	pokemonName := args[0]

	url := "https://pokeapi.co/api/v2/pokemon/" + pokemonName

	var pokemon Pokemon
	_, err := processURL(url, &pokemon)
	if err != nil {
		return err
	}
	fmt.Printf("- Throwing a Pokeball at %s...\n", pokemonName)
	fmt.Printf("Pokemon: %s, Height: %v\n", pokemonName, pokemon.Height)

	if calculateChance(pokemon) {
		fmt.Printf("%s was caught!\n", pokemonName)

		if cfg.pokedex == nil {
			cfg.pokedex = make(map[string]Pokemon)
		}
		cfg.pokedex[pokemonName] = pokemon
	} else {
		fmt.Printf("%s escaped!\n", pokemonName)
	}
	return nil
}

func commandInspect(cfg *Config, args ...string) error {
	if len(args) != 1 {
		return fmt.Errorf("you must enter a pokemon name")
	}

	pokemonName := args[0]
	caughtPokemon := cfg.pokedex

	pokemon, ok := caughtPokemon[pokemonName]
	if !ok {
		return fmt.Errorf("you haven't caught this pokemon")
	} else {
		fmt.Printf("Name: %s\n", pokemon.Name)
		fmt.Printf("Height: %v\n", pokemon.Height)
		fmt.Printf("Base Experience: %v\n", pokemon.BaseExperience)
	}
	return nil
}

func commandPokedex(cfg *Config, args ...string) error {
	caughtPokemon := cfg.pokedex

	if len(caughtPokemon) >= 1 {
		fmt.Printf("Your Pokedex:\n")
		for pokemon := range caughtPokemon {
			fmt.Printf("- %s\n", caughtPokemon[pokemon].Name)
		}
	} else {
		return fmt.Errorf("you haven't caught any pokemon")
	}
	return nil
}

var commands = map[string]cliCommand{
	"exit": {
		name:        "exit",
		description: "Exit the PokeDex",
		callback:    commandExit,
	},
	"help": {
		name:        "help",
		description: "Displays a help message",
		callback:    commandHelp,
	},
	"map": {
		name:        "map",
		description: "Fetches and displays location areas from the PokeAPI",
		callback:    commandMap,
	},
	"explore": {
		name:        "explore",
		description: "Lists pokemon in location area",
		callback:    commandExplore,
	},
	"catch": {
		name:        "catch",
		description: "Catches selected pokemon",
		callback:    commandCatch,
	},
	"inspect": {
		name:        "inspect",
		description: "Gives pokemon information",
		callback:    commandInspect,
	},
	"pokedex": {
		name:        "pokedex",
		description: "Displays pokedex information",
		callback:    commandPokedex,
	},
}

func cleanInput(text string) []string {
	words := []string{}
	currentWord := ""
	for _, char := range text {
		if char == ' ' {
			if currentWord != "" {
				words = append(words, currentWord)
				currentWord = ""
			}
		} else {
			currentWord += string(char)
		}
	}
	if currentWord != "" {
		words = append(words, currentWord)
	}
	return words
}
