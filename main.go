package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

type cliCommand struct {
	name        string
	description string
	callback    func(*Config) error
}

type Config struct {
	Next     *string
	Previous *string
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

func main() {
	input := bufio.NewScanner(os.Stdin)
	cfg := &Config{}

	for {
		fmt.Print("PokeDex > ")
		userInput := ""
		if input.Scan() {
			userInput = input.Text()
		}
		if userInput == "exit" {
			commandExit(cfg)
		}
		if userInput == "help" {
			commandHelp(cfg)
		}
		if userInput == "map" {
			commandMap(cfg)
		}
		if userInput == "mapb" {
			commandMapb(cfg)
		}
	}
}

func commandExit(cfg *Config) error {
	fmt.Print("Closing the Pokedex... Goodbye!\n")
	os.Exit(0)
	return nil
}

func commandHelp(cfg *Config) error {
	fmt.Println("\nWelcome to the Pokedex!")
	fmt.Println("Usage:")
	fmt.Println("help: displays this help message")
	fmt.Println("exit: exits the PokeDex")
	return nil
}

func commandMap(cfg *Config) error {
	var url string

	if cfg.Next != nil && *cfg.Next != "" {
		url = *cfg.Next
	} else {
		fmt.Println("You're on the first page")
		url = "https://pokeapi.co/api/v2/location-area/?limit=20"
	}
	res, err := http.Get(url)
	if err != nil {
		fmt.Println("Error fetching location areas:", err)
		return err
	}

	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return err
	}

	if res.StatusCode != http.StatusOK {
		fmt.Println("Failed to fetch location areas, status code:", res.StatusCode)
		return fmt.Errorf("failed to fetch location areas, status code: %d", res.StatusCode)
	}
	var locationData locationAreaResponse

	if err = json.Unmarshal(body, &locationData); err != nil {
		fmt.Println("Error unmarshalling response:", err)
		return err
	}
	cfg.Next = locationData.Next

	for _, area := range locationData.Results {
		fmt.Printf("%v\n", area.Name)
	}

	return nil
}

func commandMapb(cfg *Config) error {
	var url string

	if cfg.Previous != nil && *cfg.Previous != "" {
		url = *cfg.Previous
	} else {
		url = "https://pokeapi.co/api/v2/location-area/?limit=20"
	}
	res, err := http.Get(url)
	if err != nil {
		fmt.Println("Error fetching location areas:", err)
		return err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return err
	}

	if res.StatusCode != http.StatusOK {
		fmt.Println("Failed to fetch location areas, status code:", res.StatusCode)
		return fmt.Errorf("failed to fetch location areas, status code: %d", res.StatusCode)
	}
	var locationData locationAreaResponse

	if err = json.Unmarshal(body, &locationData); err != nil {
		fmt.Println("Error unmarshalling response:", err)
		return err
	}
	cfg.Previous = locationData.Previous

	for _, area := range locationData.Results {
		fmt.Printf("%v\n", area.Name)
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
