# commit-go - AI-Powered Git Commits

> _"AI wrote the code. Let it write the commit. No brain use !"_

**commit-go** is a Go-based Command Line Interface (CLI) that analyzes your Git
changes (`git diff`) and uses Artificial Intelligence to generate clear,
concise, and **Conventional Commits** compliant messages.

## Key Features

- **Multi-AI (Provider Agnostic):** Compatible with the best LLMs on the market
  (Gemini, Anthropic Claude, OpenAI, Mistral...). Switch from one to another
  with a simple config change.

* **Interactive Terminal UI:** Don't just blindly accept what the AI spits out.
  An interactive menu (powered by `charmbracelet/huh`) allows you to:
  - **Apply** the generated commit instantly.
  - **Edit** the message manually if the AI was _almost_ right.
  - **Retry** and generate a brand new proposal.
  - **Cancel** the operation in a keystroke.
* **Blazing Fast & Portable:** Written in Go, this CLI compiles down to a single
  binary. No heavy dependencies (Node.js, Python), just a lightweight executable
  that runs anywhere (Linux, macOS, Windows).
* **Simple Configuration:** Easy and centralized management of your API keys and
  preferences (powered by `spf13/viper`).

## Tech Stack

- **Language:** [Go (Golang)](https://go.dev/)
- **CLI Framework:** [Cobra](https://github.com/spf13/cobra)
- **Configuration:** [Viper](https://github.com/spf13/viper)
- **Terminal UI:** [Huh](https://github.com/charmbracelet/huh) by Charm

## Philosophy

This project was built with the **vibe coding** era in mind: the goal is to
provide the absolute smoothest Developer Experience (DX) by removing the
friction of writing commit messages, all while keeping you in full control of
your Git history.
