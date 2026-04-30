package ui

import (
	"fmt"
	"log"

	"github.com/charmbracelet/huh"
	"github.com/user/commit-go/pkg/ai"
	"github.com/user/commit-go/pkg/git"
)

func HandleCommitMenu(initialMessage string, diff string, provider ai.AIProvider) {
	currentMessage := initialMessage

	for {
		var action string
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewNote().
					Title("Generated Commit Message").
					Description(currentMessage),
				huh.NewSelect[string]().
					Title("Action").
					Options(
						huh.NewOption("Apply", "apply"),
						huh.NewOption("Edit", "edit"),
						huh.NewOption("Regenerate", "regenerate"),
						huh.NewOption("Cancel", "cancel"),
					).
					Value(&action),
			),
		)

		err := form.Run()
		if err != nil {
			log.Fatal(err)
		}

		switch action {
		case "apply":
			if err := git.ExecuteCommit(currentMessage); err != nil {
				fmt.Printf("Error committing: %v\n", err)
			}
			return
		case "edit":
			var newMessage string
			editForm := huh.NewForm(
				huh.NewGroup(
					huh.NewInput().
						Title("Edit Commit Message").
						Value(&newMessage).
						Placeholder(currentMessage),
				),
			)
			if err := editForm.Run(); err == nil && newMessage != "" {
				currentMessage = newMessage
			}
		case "regenerate":
			fmt.Println("Regenerating...")
			msg, err := provider.GenerateCommit(diff)
			if err != nil {
				fmt.Printf("Error regenerating: %v\n", err)
			} else {
				currentMessage = msg
			}
		case "cancel":
			fmt.Println("Commit cancelled.")
			return
		}
	}
}
