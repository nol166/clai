package main

import "fmt"

const asciiLogo = `
  ██████╗██╗      █████╗ ██╗
 ██╔════╝██║     ██╔══██╗██║
 ██║     ██║     ███████║██║
 ██║     ██║     ██╔══██║██║
 ╚██████╗███████╗██║  ██║██║
  ╚═════╝╚══════╝╚═╝  ╚═╝╚═╝`

func printLogo() {
	fmt.Printf("\033[38;5;99m%s\033[0m\n", asciiLogo)
	fmt.Print("  \033[2mask your terminal anything\033[0m\n\n")
}

func printHelp() {
	printLogo()
	fmt.Print(`Usage:
  clai <query>               ask anything
  clai -c <query>            ask and copy response to clipboard

Config:
  clai config                interactive setup
  clai config list           show current config
  clai config set <k> <v>    set a value  (provider/model/api-key/base-url)
  clai config clipboard      toggle always-copy-to-clipboard

Models:
  clai models list           list available models for current provider

Flags:
  -c, --copy                 copy response to clipboard
  -v, --version              show version
  -h, --help                 show this help

Providers:
  openai    OPENAI_API_KEY or CLAI_API_KEY
  anthropic ANTHROPIC_API_KEY or CLAI_API_KEY
  litellm   CLAI_BASE_URL + CLAI_API_KEY
  ollama    no key needed, defaults to http://localhost:11434/v1

`)
}
