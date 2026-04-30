package ui

import (
	"fmt"
	"log"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
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
			msg, err := GenerateWithSpinner(diff, provider)
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

// spinnerModel represents the state of the spinner TUI
type spinnerModel struct {
	spinner  spinner.Model
	quitting bool
	err      error
	msg      string
	diff     string
	provider ai.AIProvider
	done     chan bool
}

type gotMsg struct {
	msg string
	err error
}

func (m spinnerModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		func() tea.Msg {
			msg, err := m.provider.GenerateCommit(m.diff)
			return gotMsg{msg: msg, err: err}
		},
	)
}

func (m spinnerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		default:
			return m, nil
		}
	case gotMsg:
		m.msg = msg.msg
		m.err = msg.err
		m.quitting = true
		return m, tea.Quit
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	default:
		return m, nil
	}
}

func (m spinnerModel) View() string {
	if m.err != nil {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render(fmt.Sprintf("Error: %v", m.err))
	}
	if m.quitting {
		return ""
	}
	str := fmt.Sprintf("\n %s Generating message using %s...\n\n", m.spinner.View(), m.provider.GetName())
	return str
}

func GenerateWithSpinner(diff string, provider ai.AIProvider) (string, error) {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	m := spinnerModel{
		spinner:  s,
		diff:     diff,
		provider: provider,
	}

	p := tea.NewProgram(m)
	res, err := p.Run()
	if err != nil {
		return "", err
	}

	finalModel := res.(spinnerModel)
	if finalModel.err != nil {
		return "", finalModel.err
	}

	return finalModel.msg, nil
}
