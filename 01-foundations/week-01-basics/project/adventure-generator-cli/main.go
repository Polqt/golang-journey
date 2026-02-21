package main

import (
	"flag"
	"fmt"
	"math/rand"
)

var count = flag.Int("count", 1, "Number of adventures to generate")
var adventureType = flag.String("type", "fantasy", "Type of adventure (fantasy, sci-fi, mystery)")
var protagonistName = flag.String("name", "Jepoy", "Name of the protagonist")

func generateAdventure(adventureType string, protagonistName string, templates map[string][]string) string {
	templatesForType := templates[adventureType]
	selectedTemplate := templatesForType[rand.Intn(len(templatesForType))] 
	return fmt.Sprintf(selectedTemplate, protagonistName)	
}

func printAdventures(adventures []string) {
	for i, adventure := range adventures {
		fmt.Printf("Adventure %d: %s\n", i+1, adventure)
	}
}


func main() {
	flag.Parse()

	adventureTemplates := map[string][]string {
		"fantasy": {
			"%s must recover the lost Sword of Dawn from the haunted forest.",
			"%s is tasked with rescuing the dragon's egg from the mountain cave.",
			"%s embarks on a journey to find the ancient wizard's hidden tower.",
			"%s must unite the warring kingdoms to face a common enemy.",
			"%s discovers a magical artifact that could change the fate of the realm.",
			"%s is chosen to be the guardian of a mystical portal to another world.",
		}, 
		"sci-fi": {
			"%s must navigate through a space station overrun by aliens.",
			"%s is on a mission to find a new habitable planet for humanity.",
			"%s discovers a hidden message from an ancient alien civilization.",
			"%s must prevent a rogue AI from taking over the galaxy.",
			"%s is part of a crew exploring a mysterious wormhole.",
			"%s must lead a rebellion against a tyrannical interstellar empire.",
		},
		"mystery": {
			"%s must solve the case of the missing artifact in the old mansion.",
			"%s is investigating a series of strange disappearances in the small town.",
			"%s uncovers a hidden diary that reveals dark secrets about the town's history.",
			"%s must piece together clues to catch a cunning thief.",
			"%s is hired to find a lost heirloom with a mysterious past.",
			"%s must navigate a web of lies to uncover the truth behind a high-profile scandal.",
			"%s stumbles upon a secret society while investigating a local legend.",
		},
		"horror": {
			"%s must survive the night in a haunted asylum.",
			"%s is trapped in a cabin with a lurking monster outside.",
			"%s discovers an ancient curse that threatens their family.",
			"%s must escape a town overrun by zombies.",
			"%s is being hunted by a supernatural entity in the woods.",
			"%s uncovers a dark ritual being performed in their neighborhood.",
			"%s must confront their deepest fears to break a haunting spell.",
		},
		"adventure": {
			"%s embarks on a quest to find the lost city of gold.",
			"%s must navigate through uncharted jungles to find a hidden treasure.",
			"%s is stranded on a deserted island and must find a way to survive.",
			"%s leads an expedition to explore ancient ruins filled with traps.",
			"%s must cross treacherous mountains to deliver a vital message.",
			"%s is on a mission to rescue a kidnapped explorer from hostile territory.",
			"%s discovers a secret map that leads to a legendary artifact.",
		},
		"romance": {
			"%s finds love in the most unexpected place during a summer vacation.",
			"%s must choose between two suitors vying for their heart.",
			"%s reunites with a childhood friend and sparks fly.",
			"%s navigates the challenges of a long-distance relationship.",
			"%s discovers a secret admirer leaving mysterious notes.",
			"%s must overcome personal fears to confess their love.",
			"%s embarks on a romantic adventure across Europe.",
			"%s finds love while solving a mystery together.",
		},
		"historical": {
			"%s is caught in the midst of a revolution and must choose a side.",
			"%s uncovers a secret that could change the course of history.",
			"%s navigates the challenges of life in a medieval kingdom.",
			"%s must protect a priceless artifact during wartime.",
			"%s is a spy in ancient Rome trying to prevent a conspiracy.",
			"%s leads a group of settlers to establish a new colony.",
			"%s must solve a mystery in Victorian London.",
		},
		"superhero": {
			"%s discovers their superpowers and must learn to control them.",
			"%s faces off against a formidable villain threatening the city.",
			"%s joins a team of superheroes to save the world from destruction.",
			"%s must protect their secret identity while fighting crime.",
			"%s uncovers a plot to unleash chaos and must stop it.",
			"%s struggles with the responsibilities of being a hero.",
			"%s must rally other heroes to face a common enemy.",
		},
		"detective": {
			"%s must crack the code to catch a cunning criminal mastermind.",
			"%s is on the trail of a jewel thief in a bustling metropolis.",
			"%s uncovers a conspiracy while investigating a high-profile case.",
			"%s must navigate the city's underworld to find a missing person.",
			"%s uses their keen observation skills to solve a baffling mystery.",
			"%s must outsmart a rival detective to claim the glory.",
			"%s is hired to protect a witness in a dangerous trial.",
		},
		"dystopian": {
			"%s fights against an oppressive regime in a dystopian future.",
			"%s leads a rebellion to restore freedom to their society.",
			"%s uncovers hidden truths about the world they live in.",
			"%s must navigate a dangerous landscape to find a safe haven.",
			"%s is part of an underground movement resisting control.",
			"%s discovers a way to bring hope to a bleak world.",
			"%s must infiltrate the ruling class to gather crucial information.",
		},
		"mythology": {
			"%s embarks on a quest to retrieve a stolen artifact from the gods.",
			"%s must navigate the challenges set by mythical creatures to prove their worth.",
			"%s is chosen by the gods to undertake a perilous journey.",
			"%s must unite warring factions of mythical beings to face a common threat.",
			"%s discovers their divine heritage and the powers that come with it.",
			"%s must solve ancient riddles to unlock the secrets of the gods.",
			"%s faces trials set by the gods to earn their favor.",
		},
		"noir": {
			"%s is a hard-boiled detective unraveling a web of deceit in the city.",
			"%s must navigate the dark underworld to solve a high-profile case.",
			"%s is drawn into a dangerous game of cat and mouse with a cunning criminal.",
			"%s must protect a femme fatale while uncovering the truth.",
			"%s battles personal demons while seeking justice in a corrupt city.",
			"%s uncovers a conspiracy that reaches the highest levels of power.",
			"%s must outwit a rival detective to solve a baffling mystery.",
		},
		"western": {
			"%s is a lone gunslinger seeking justice in a lawless town.",
			"%s must protect a small settlement from a band of outlaws.",
			"%s embarks on a journey across the frontier to find a lost treasure.",
			"%s is caught in a feud between rival ranchers and must find a way to bring peace.",
			"%s discovers a hidden gold mine and must defend it from greedy prospectors.",
			"%s is hired to track down a notorious bandit terrorizing the region.",
			"%s must navigate the challenges of life in the Wild West to build a new life.",
		},
	}

	// TODO: Validate user input (e.g., check if *adventureType exists in adventureTemplates, *count > 0)
	if _, exists := adventureTemplates[*adventureType]; !exists {
		fmt.Printf("Error: Adventure type '%s' not recognized.\n", *adventureType)
		return
	}

	// TODO: Create a slice to store generated adventures
	adventures := []string{}

	// TODO: Use a loop to generate the requested number of adventures
	//   - Call generateAdventure for each adventure
	//   - Append each result to the adventures slice
	for i := 0; i < *count; i++ {
		adventure := generateAdventure(*adventureType, *protagonistName, adventureTemplates)
		adventures = append(adventures, adventure)
	}

	// TODO: Print all generated adventures using printAdventures
	printAdventures(adventures)

}
