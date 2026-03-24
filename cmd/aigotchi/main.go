package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/egoomoy/aigotchi/internal/collector"
	"github.com/egoomoy/aigotchi/internal/engine"
	"github.com/egoomoy/aigotchi/internal/pet"
	"github.com/egoomoy/aigotchi/internal/renderer"
	"github.com/egoomoy/aigotchi/internal/storage"
	"github.com/egoomoy/aigotchi/internal/tui"
)

var version = "dev"

const (
	dataDir   = "~/.aigotchi"
	petFile   = "pet.json"
	stateFile = "state.json"
	eventsFile = "events.jsonl"
)

func expandHome(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[2:])
	}
	return path
}

func openStore() (*storage.Store, error) {
	dir := expandHome(dataDir)
	return storage.NewStore(dir)
}

func loadPetAndState(store *storage.Store) (pet.Pet, engine.State, error) {
	var p pet.Pet
	var s engine.State

	if err := store.ReadJSON(petFile, &p); err != nil {
		return p, s, fmt.Errorf("read pet: %w (run 'aigotchi init' first)", err)
	}
	if err := store.ReadJSON(stateFile, &s); err != nil {
		return p, s, fmt.Errorf("read state: %w (run 'aigotchi init' first)", err)
	}
	return p, s, nil
}

func savePetAndState(store *storage.Store, p pet.Pet, s engine.State) error {
	s.LastUpdated = time.Now()
	if err := store.WriteJSON(petFile, p); err != nil {
		return fmt.Errorf("write pet: %w", err)
	}
	if err := store.WriteJSON(stateFile, s); err != nil {
		return fmt.Errorf("write state: %w", err)
	}
	return nil
}

var rootCmd = &cobra.Command{
	Use:   "aigotchi",
	Short: "Your AI-powered virtual pet",
	Long:  "aigotchi is a virtual pet that grows with your Claude Code usage.",
	RunE: func(cmd *cobra.Command, args []string) error {
		store, err := openStore()
		if err != nil {
			return err
		}
		if !store.Exists(petFile) || !store.Exists(stateFile) {
			fmt.Fprintln(os.Stderr, "No pet found. Run 'aigotchi init' first.")
			os.Exit(1)
		}
		m, err := tui.NewModel(store)
		if err != nil {
			return err
		}
		p := tea.NewProgram(m, tea.WithAltScreen())
		_, runErr := p.Run()
		return runErr
	},
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Create a new pet and register Claude Code hook",
	RunE: func(cmd *cobra.Command, args []string) error {
		store, err := openStore()
		if err != nil {
			return err
		}

		if !store.Exists(petFile) {
			p := pet.NewPet("Mochi")
			if err := store.WriteJSON(petFile, p); err != nil {
				return fmt.Errorf("write pet: %w", err)
			}
			fmt.Println("Created pet.json")
		} else {
			fmt.Println("pet.json already exists, skipping")
		}

		if !store.Exists(stateFile) {
			s := engine.NewState()
			if err := store.WriteJSON(stateFile, s); err != nil {
				return fmt.Errorf("write state: %w", err)
			}
			fmt.Println("Created state.json")
		} else {
			fmt.Println("state.json already exists, skipping")
		}

		if err := registerHook(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not register hook: %v\n", err)
		} else {
			fmt.Println("Registered Claude Code Stop hook")
		}

		return nil
	},
}

// hookEntry represents one hook entry in Claude Code settings.
type hookEntry struct {
	Type    string `json:"type"`
	Command string `json:"command"`
}

// hooksConfig mirrors the relevant part of Claude Code's settings.json.
type hooksConfig struct {
	Hooks map[string][]hookEntry `json:"hooks"`
}

func registerHook() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	settingsPath := filepath.Join(home, ".claude", "settings.json")

	// Read existing settings or start fresh
	var cfg map[string]json.RawMessage
	if data, err := os.ReadFile(settingsPath); err == nil {
		if err := json.Unmarshal(data, &cfg); err != nil {
			cfg = make(map[string]json.RawMessage)
		}
	} else {
		cfg = make(map[string]json.RawMessage)
	}

	// Parse existing hooks
	var hooks map[string][]map[string]interface{}
	if raw, ok := cfg["hooks"]; ok {
		if err := json.Unmarshal(raw, &hooks); err != nil {
			hooks = make(map[string][]map[string]interface{})
		}
	} else {
		hooks = make(map[string][]map[string]interface{})
	}

	// The aigotchi collect command for the hook
	aigotchiCmd := "aigotchi collect --session-id $SESSION_ID --cwd $PWD"

	// Check if the hook already exists in Stop hooks
	stopHooks := hooks["Stop"]
	for _, h := range stopHooks {
		if cmd, ok := h["command"].(string); ok && cmd == aigotchiCmd {
			fmt.Println("Hook already registered")
			return nil
		}
	}

	// Append the new hook
	newHook := map[string]interface{}{
		"type":    "command",
		"command": aigotchiCmd,
	}
	hooks["Stop"] = append(stopHooks, newHook)

	// Marshal back
	hooksRaw, err := json.Marshal(hooks)
	if err != nil {
		return err
	}
	cfg["hooks"] = hooksRaw

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(settingsPath), 0755); err != nil {
		return err
	}

	return os.WriteFile(settingsPath, data, 0644)
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show one-line pet status",
	RunE: func(cmd *cobra.Command, args []string) error {
		store, err := openStore()
		if err != nil {
			return err
		}
		p, s, err := loadPetAndState(store)
		if err != nil {
			return err
		}

		xp := engine.CurrentXP(s)
		line := renderer.RenderStatusLine(&p, s.Hunger, s.Happiness, s.Health, xp)
		fmt.Println(line)
		return nil
	},
}

var feedCmd = &cobra.Command{
	Use:   "feed",
	Short: "Feed your pet (costs tokens)",
	RunE: func(cmd *cobra.Command, args []string) error {
		store, err := openStore()
		if err != nil {
			return err
		}
		p, s, err := loadPetAndState(store)
		if err != nil {
			return err
		}

		// Apply time delta first
		delta := time.Since(s.LastUpdated)
		s = engine.ApplyTimeDelta(s, delta)

		newState, err := engine.Feed(s)
		if err != nil {
			return fmt.Errorf("cannot feed: %w", err)
		}
		s = newState

		if err := savePetAndState(store, p, s); err != nil {
			return err
		}

		fmt.Printf("Fed %s! Hunger: %d\n", p.Name, s.Hunger)
		return nil
	},
}

var playCmd = &cobra.Command{
	Use:   "play",
	Short: "Play with your pet",
	RunE: func(cmd *cobra.Command, args []string) error {
		store, err := openStore()
		if err != nil {
			return err
		}
		p, s, err := loadPetAndState(store)
		if err != nil {
			return err
		}

		// Apply time delta first
		delta := time.Since(s.LastUpdated)
		s = engine.ApplyTimeDelta(s, delta)

		s = engine.Play(s, false)

		if err := savePetAndState(store, p, s); err != nil {
			return err
		}

		fmt.Printf("Played with %s! Happiness: %d\n", p.Name, s.Happiness)
		return nil
	},
}

// eventRecord is one line written to events.jsonl.
type eventRecord struct {
	Timestamp    time.Time `json:"timestamp"`
	SessionID    string    `json:"session_id,omitempty"`
	TotalTokens  int64     `json:"total_tokens"`
	MessageCount int       `json:"message_count"`
	Model        string    `json:"model,omitempty"`
	NewOffset    int64     `json:"new_offset"`
}

var collectCmd = &cobra.Command{
	Use:   "collect",
	Short: "Parse transcript and update token stats",
	RunE: func(cmd *cobra.Command, args []string) error {
		sessionID, _ := cmd.Flags().GetString("session-id")
		cwd, _ := cmd.Flags().GetString("cwd")
		_ = cwd // may be used for future context

		transcriptPath, err := findTranscript(sessionID)
		if err != nil {
			// If we can't find transcript, silently exit (hook runs in background)
			return nil
		}

		store, err := openStore()
		if err != nil {
			return err
		}

		// Determine offset from last event
		var fromOffset int64
		lines, readErr := store.ReadLines(eventsFile)
		if readErr == nil && len(lines) > 0 {
			last := lines[len(lines)-1]
			var lastEvent eventRecord
			if err := json.Unmarshal(last, &lastEvent); err == nil {
				// Only use offset if same session
				if lastEvent.SessionID == sessionID {
					fromOffset = lastEvent.NewOffset
				}
			}
		}

		result, err := collector.ParseTranscript(transcriptPath, fromOffset)
		if err != nil {
			return fmt.Errorf("parse transcript: %w", err)
		}

		if result.TotalTokens == 0 {
			return nil
		}

		// Write event to events.jsonl
		rec := eventRecord{
			Timestamp:    time.Now(),
			SessionID:    sessionID,
			TotalTokens:  result.TotalTokens,
			MessageCount: result.MessageCount,
			Model:        result.Model,
			NewOffset:    result.NewOffset,
		}
		recData, err := json.Marshal(rec)
		if err != nil {
			return fmt.Errorf("marshal event: %w", err)
		}
		if err := store.AppendLine(eventsFile, recData); err != nil {
			return fmt.Errorf("write event: %w", err)
		}

		// Update state TotalTokensEarned
		p, s, err := loadPetAndState(store)
		if err != nil {
			return err
		}

		s.TotalTokensEarned += result.TotalTokens

		// Check for evolution after token update
		evolved, newPet, newState := engine.CheckEvolution(p, s)
		if evolved {
			p = newPet
			s = newState
			fmt.Printf("%s evolved to %s!\n", p.Name, p.Stage)
		}

		if err := savePetAndState(store, p, s); err != nil {
			return err
		}

		fmt.Printf("Collected %d tokens (%d messages)\n", result.TotalTokens, result.MessageCount)
		return nil
	},
}

// findTranscript searches ~/.claude/projects/ for a JSONL file matching sessionID.
func findTranscript(sessionID string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	projectsDir := filepath.Join(home, ".claude", "projects")
	if _, err := os.Stat(projectsDir); os.IsNotExist(err) {
		return "", fmt.Errorf("projects dir not found: %s", projectsDir)
	}

	// Walk through project subdirectories looking for the session file
	var found string
	err = filepath.Walk(projectsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip errors
		}
		if info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".jsonl") {
			return nil
		}
		name := strings.TrimSuffix(filepath.Base(path), ".jsonl")
		if sessionID == "" || name == sessionID {
			found = path
			return filepath.SkipAll
		}
		return nil
	})

	if err != nil && err != filepath.SkipAll {
		return "", err
	}

	if found == "" {
		return "", fmt.Errorf("transcript not found for session %q", sessionID)
	}
	return found, nil
}

var nameCmd = &cobra.Command{
	Use:   "name <name>",
	Short: "Set the pet's name",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		store, err := openStore()
		if err != nil {
			return err
		}
		p, s, err := loadPetAndState(store)
		if err != nil {
			return err
		}

		oldName := p.Name
		p.Name = args[0]

		if err := savePetAndState(store, p, s); err != nil {
			return err
		}

		fmt.Printf("Renamed %s to %s\n", oldName, p.Name)
		return nil
	},
}

func init() {
	collectCmd.Flags().String("session-id", "", "session ID")
	collectCmd.Flags().String("cwd", "", "working directory")

	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(feedCmd)
	rootCmd.AddCommand(playCmd)
	rootCmd.AddCommand(collectCmd)
	rootCmd.AddCommand(nameCmd)
	rootCmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Print aigotchi version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("aigotchi", version)
		},
	})
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
