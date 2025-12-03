package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/asheshgoplani/agent-deck/internal/session"
	"github.com/asheshgoplani/agent-deck/internal/ui"
)

const Version = "0.1.0"

func main() {
	// Handle subcommands
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "version", "--version", "-v":
			fmt.Printf("Agent Deck v%s\n", Version)
			return
		case "help", "--help", "-h":
			printHelp()
			return
		case "add":
			handleAdd(os.Args[2:])
			return
		case "list", "ls":
			handleList(os.Args[2:])
			return
		case "remove", "rm":
			handleRemove(os.Args[2:])
			return
		}
	}

	// Check if tmux is available
	if _, err := exec.LookPath("tmux"); err != nil {
		fmt.Println("Error: tmux not found in PATH")
		fmt.Println("\nAgent Deck requires tmux. Install with:")
		fmt.Println("  brew install tmux")
		os.Exit(1)
	}

	p := tea.NewProgram(
		ui.NewHome(),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

// handleAdd adds a new session from CLI
func handleAdd(args []string) {
	fs := flag.NewFlagSet("add", flag.ExitOnError)
	title := fs.String("title", "", "Session title (defaults to folder name)")
	titleShort := fs.String("t", "", "Session title (short)")
	group := fs.String("group", "", "Group path (defaults to parent folder)")
	groupShort := fs.String("g", "", "Group path (short)")
	command := fs.String("cmd", "", "Command to run (e.g., 'claude', 'aider')")
	commandShort := fs.String("c", "", "Command to run (short)")

	fs.Usage = func() {
		fmt.Println("Usage: agent-deck add <path> [options]")
		fmt.Println()
		fmt.Println("Add a new session to Agent Deck.")
		fmt.Println()
		fmt.Println("Arguments:")
		fmt.Println("  <path>    Project directory (use '.' for current directory)")
		fmt.Println()
		fmt.Println("Options:")
		fs.PrintDefaults()
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  agent-deck add .")
		fmt.Println("  agent-deck add /path/to/project")
		fmt.Println("  agent-deck add -t \"My Project\" -g \"work\" .")
		fmt.Println("  agent-deck add -c claude .")
	}

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	// Get path argument
	path := fs.Arg(0)
	if path == "" {
		fmt.Println("Error: path is required")
		fmt.Println("Usage: agent-deck add <path> [options]")
		os.Exit(1)
	}

	// Resolve path
	if path == "." {
		var err error
		path, err = os.Getwd()
		if err != nil {
			fmt.Printf("Error: failed to get current directory: %v\n", err)
			os.Exit(1)
		}
	} else {
		var err error
		path, err = filepath.Abs(path)
		if err != nil {
			fmt.Printf("Error: failed to resolve path: %v\n", err)
			os.Exit(1)
		}
	}

	// Verify path exists and is a directory
	info, err := os.Stat(path)
	if err != nil {
		fmt.Printf("Error: path does not exist: %s\n", path)
		os.Exit(1)
	}
	if !info.IsDir() {
		fmt.Printf("Error: path is not a directory: %s\n", path)
		os.Exit(1)
	}

	// Merge short and long flags
	sessionTitle := mergeFlags(*title, *titleShort)
	sessionGroup := mergeFlags(*group, *groupShort)
	sessionCommand := mergeFlags(*command, *commandShort)

	// Default title to folder name
	if sessionTitle == "" {
		sessionTitle = filepath.Base(path)
	}

	// Load existing sessions
	storage, err := session.NewStorage()
	if err != nil {
		fmt.Printf("Error: failed to initialize storage: %v\n", err)
		os.Exit(1)
	}

	instances, groups, err := storage.LoadWithGroups()
	if err != nil {
		fmt.Printf("Error: failed to load sessions: %v\n", err)
		os.Exit(1)
	}

	// Check for duplicate (same path)
	for _, inst := range instances {
		if inst.ProjectPath == path {
			fmt.Printf("Session already exists: %s (%s)\n", inst.Title, inst.ID)
			os.Exit(0)
		}
	}

	// Create new instance (without starting tmux)
	var newInstance *session.Instance
	if sessionGroup != "" {
		newInstance = session.NewInstanceWithGroup(sessionTitle, path, sessionGroup)
	} else {
		newInstance = session.NewInstance(sessionTitle, path)
	}

	// Set command if provided
	if sessionCommand != "" {
		newInstance.Command = sessionCommand
		// Detect tool from command
		newInstance.Tool = detectTool(sessionCommand)
	}

	// Add to instances
	instances = append(instances, newInstance)

	// Rebuild group tree and save
	groupTree := session.NewGroupTreeWithGroups(instances, groups)
	// Ensure the session's group exists
	if newInstance.GroupPath != "" {
		groupTree.CreateGroup(newInstance.GroupPath)
	}

	if err := storage.SaveWithGroups(instances, groupTree); err != nil {
		fmt.Printf("Error: failed to save session: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Added session: %s\n", sessionTitle)
	fmt.Printf("  Path:  %s\n", path)
	fmt.Printf("  Group: %s\n", newInstance.GroupPath)
	fmt.Printf("  ID:    %s\n", newInstance.ID)
	if sessionCommand != "" {
		fmt.Printf("  Cmd:   %s\n", sessionCommand)
	}
}

// handleList lists all sessions
func handleList(args []string) {
	fs := flag.NewFlagSet("list", flag.ExitOnError)
	jsonOutput := fs.Bool("json", false, "Output as JSON")

	fs.Usage = func() {
		fmt.Println("Usage: agent-deck list [options]")
		fmt.Println()
		fmt.Println("List all sessions.")
		fmt.Println()
		fmt.Println("Options:")
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	storage, err := session.NewStorage()
	if err != nil {
		fmt.Printf("Error: failed to initialize storage: %v\n", err)
		os.Exit(1)
	}

	instances, _, err := storage.LoadWithGroups()
	if err != nil {
		fmt.Printf("Error: failed to load sessions: %v\n", err)
		os.Exit(1)
	}

	if len(instances) == 0 {
		fmt.Println("No sessions found.")
		return
	}

	if *jsonOutput {
		// JSON output for scripting
		type sessionJSON struct {
			ID          string    `json:"id"`
			Title       string    `json:"title"`
			Path        string    `json:"path"`
			Group       string    `json:"group"`
			Tool        string    `json:"tool"`
			Command     string    `json:"command,omitempty"`
			CreatedAt   time.Time `json:"created_at"`
		}
		sessions := make([]sessionJSON, len(instances))
		for i, inst := range instances {
			sessions[i] = sessionJSON{
				ID:        inst.ID,
				Title:     inst.Title,
				Path:      inst.ProjectPath,
				Group:     inst.GroupPath,
				Tool:      inst.Tool,
				Command:   inst.Command,
				CreatedAt: inst.CreatedAt,
			}
		}
		output, _ := json.MarshalIndent(sessions, "", "  ")
		fmt.Println(string(output))
		return
	}

	// Table output
	fmt.Printf("%-20s %-15s %-40s %s\n", "TITLE", "GROUP", "PATH", "ID")
	fmt.Println(strings.Repeat("-", 100))
	for _, inst := range instances {
		title := truncate(inst.Title, 20)
		group := truncate(inst.GroupPath, 15)
		path := truncate(inst.ProjectPath, 40)
		fmt.Printf("%-20s %-15s %-40s %s\n", title, group, path, inst.ID[:12])
	}
	fmt.Printf("\nTotal: %d sessions\n", len(instances))
}

// handleRemove removes a session by ID or title
func handleRemove(args []string) {
	fs := flag.NewFlagSet("remove", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Println("Usage: agent-deck remove <id|title>")
		fmt.Println()
		fmt.Println("Remove a session by ID or title.")
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  agent-deck remove abc12345")
		fmt.Println("  agent-deck remove \"My Project\"")
	}

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	identifier := fs.Arg(0)
	if identifier == "" {
		fmt.Println("Error: session ID or title is required")
		fs.Usage()
		os.Exit(1)
	}

	storage, err := session.NewStorage()
	if err != nil {
		fmt.Printf("Error: failed to initialize storage: %v\n", err)
		os.Exit(1)
	}

	instances, groups, err := storage.LoadWithGroups()
	if err != nil {
		fmt.Printf("Error: failed to load sessions: %v\n", err)
		os.Exit(1)
	}

	// Find and remove the session
	found := false
	var removedTitle string
	newInstances := make([]*session.Instance, 0, len(instances))
	for _, inst := range instances {
		if inst.ID == identifier || strings.HasPrefix(inst.ID, identifier) || inst.Title == identifier {
			found = true
			removedTitle = inst.Title
			// Kill tmux session if it exists
			if inst.Exists() {
				inst.Kill()
			}
		} else {
			newInstances = append(newInstances, inst)
		}
	}

	if !found {
		fmt.Printf("Error: session not found: %s\n", identifier)
		os.Exit(1)
	}

	// Rebuild group tree and save
	groupTree := session.NewGroupTreeWithGroups(newInstances, groups)

	if err := storage.SaveWithGroups(newInstances, groupTree); err != nil {
		fmt.Printf("Error: failed to save: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Removed session: %s\n", removedTitle)
}

func printHelp() {
	fmt.Printf("Agent Deck v%s\n", Version)
	fmt.Println("Terminal session manager for AI coding agents")
	fmt.Println()
	fmt.Println("Usage: agent-deck [command]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  (none)       Start the TUI")
	fmt.Println("  add <path>   Add a new session")
	fmt.Println("  list, ls     List all sessions")
	fmt.Println("  remove, rm   Remove a session")
	fmt.Println("  version      Show version")
	fmt.Println("  help         Show this help")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  agent-deck add .                      # Add current directory")
	fmt.Println("  agent-deck add /path/to/project       # Add specific path")
	fmt.Println("  agent-deck add -t \"My App\" -g work .  # With title and group")
	fmt.Println("  agent-deck add -c claude .            # With command")
	fmt.Println("  agent-deck list                       # List all sessions")
	fmt.Println("  agent-deck list -json                 # JSON output")
	fmt.Println("  agent-deck remove my-project          # Remove by title")
	fmt.Println()
	fmt.Println("Keyboard shortcuts (in TUI):")
	fmt.Println("  n          New session")
	fmt.Println("  g          New group")
	fmt.Println("  Enter      Attach to session")
	fmt.Println("  d          Delete session/group")
	fmt.Println("  m          Move session to group")
	fmt.Println("  R          Rename session/group")
	fmt.Println("  /          Search")
	fmt.Println("  Ctrl+Q     Detach from session")
	fmt.Println("  q          Quit")
}

// mergeFlags returns the non-empty value, preferring the first
func mergeFlags(long, short string) string {
	if long != "" {
		return long
	}
	return short
}

// truncate shortens a string to max length with ellipsis
func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	if max <= 3 {
		return s[:max]
	}
	return s[:max-3] + "..."
}

// detectTool determines the tool type from command
func detectTool(cmd string) string {
	cmd = strings.ToLower(cmd)
	switch {
	case strings.Contains(cmd, "claude"):
		return "claude"
	case strings.Contains(cmd, "aider"):
		return "aider"
	case strings.Contains(cmd, "gemini"):
		return "gemini"
	case strings.Contains(cmd, "codex"):
		return "codex"
	case strings.Contains(cmd, "cursor"):
		return "cursor"
	default:
		return "shell"
	}
}
