package main

import (
	"fmt"
	"os"

	mirrorconfig "mirror/internal/config"
)

func main() {
	mirrorconfig.Init()

	fmt.Println("=== LLM Config ===")
	fmt.Printf("Provider: %q\n", mirrorconfig.Conf.LLM.Provider)
	fmt.Printf("Model:    %q\n", mirrorconfig.Conf.LLM.Model)
	fmt.Printf("HasValid: %v\n", mirrorconfig.HasValidLLM())
	fmt.Printf("Resolved: %q\n", mirrorconfig.ResolveAPIKey())

	fmt.Println("\n=== Env Check ===")
	fmt.Printf("OPENAI_API_KEY: %q\n", os.Getenv("OPENAI_API_KEY"))
	fmt.Printf("ANTHROPIC_API_KEY: %q\n", os.Getenv("ANTHROPIC_API_KEY"))
	fmt.Printf("MIRROR_LLM_API_KEY: %q\n", os.Getenv("MIRROR_LLM_API_KEY"))
}
