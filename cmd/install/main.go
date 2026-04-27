package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/exp/teatest"
)

var (
	defaultPort           = 9090
	defaultLogLevel       = "info"
	defaultLogFormat      = "text"
	defaultRequestTimeout = 3600
	defaultMaxBodySize    = 100 * 1024 * 1024

	validLogLevels  = []string{"debug", "info", "warn", "error"}
	validLogFormats = []string{"text", "json"}
)

// Styles
var (
	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("13")).
			Bold(true).
			Align(lipgloss.Center)

	headerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("12")).
			Bold(true)

	labelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("8")).
			Width(25)

	hintStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("14")).
			Faint(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("1")).
			Bold(true)

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("2")).
			Bold(true)

	infoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("6"))

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("8")).
			Padding(1, 2)
)

// installConfig represents the final configuration
type installConfig struct {
	ServerPort          int
	ServerHost          string
	MaxRequestBodySize  int
	Timeout             int
	LogLevel            string
	LogFormat           string
	AltRootDir          string
	Endpoints           []endpointConfig
	SystemdService      bool
	ConfigPath          string
	BinaryPath          string
	InstallToPath       bool
	ServiceUser         string
}

type endpointConfig struct {
	Host   string
	Models []modelConfig
}

type modelConfig struct {
	ID                string
	Path              string
	ModelName         string
	Temperature       float64
	TopP              float64
	TopK              int
	Stream            bool
	RepetitionPenalty float64
}

// State represents the current step in the wizard
type State int

const (
	StateStart State = iota
	StateServerConfig
	StatePathConfig
	StateSystemdBinaryPath
	StateEndpointHost
	StateEndpointModels
	StateSystemdService
	StateSummary
	StateGenerating
	StateComplete
	StateError
	StateConfigPort
	StateConfigLogLevel
	StateConfigLogFormat
	StateModelID
	StateModelPath
	StateModelName
	StateModelTemp
	StateModelTopP
	StateModelTopK
	StateModelStream
	StateModelRepPenalty
	StateConfigDir
	StateServiceUser
)

// model is the bubbletea model for the installer
type model struct {
	state          State
	config         installConfig
	currentIndex   int
	errorMsg       string
	input          string
	width          int
	height         int
	endpoints      []endpointConfig
	models         []modelConfig
	currentEndpoint endpointConfig
	currentModel   modelConfig
}

// tickMsg is sent periodically to update the cursor
type tickMsg struct{}

// Init initializes the model
func (m model) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the model
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit

		case tea.KeyEnter:
			return m.handleEnter()

		case tea.KeyCtrlB, tea.KeyLeft:
			return m.handleBack()

		default:
			// Handle character input
			if msg.Type == tea.KeySpace {
				m.input += " "
			} else if len(msg.Runes) > 0 {
				m.input += string(msg.Runes)
			}
		}

	case tea.KeyMsg:
		// Handle backspace
		if msg.Type == tea.KeyBackspace || msg.Type == tea.KeyCtrlH {
			if len(m.input) > 0 {
				m.input = m.input[:len(m.input)-1]
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}

	return m, nil
}

func (m model) handleEnter() (model, tea.Cmd) {
	switch m.state {
	case StateStart:
		m.state = StateServerConfig
		m.errorMsg = ""
		m.config = installConfig{
			ServerHost:         "0.0.0.0",
			MaxRequestBodySize: defaultMaxBodySize,
			Timeout:            defaultRequestTimeout,
			AltRootDir:         "/lmproxy",
		}

	case StateServerConfig:
		m.state = StateConfigPort
		m.input = fmt.Sprintf("%d", defaultPort)

	case StateConfigPort:
		port := parseInt(m.input, defaultPort)
		if port <= 0 || port > 65535 {
			m.errorMsg = "Port must be between 1 and 65535"
			return m, nil
		}
		m.config.ServerPort = port
		m.state = StateConfigLogLevel
		m.errorMsg = ""

	case StateConfigLogLevel:
		level := strings.ToLower(strings.TrimSpace(m.input))
		if !contains(validLogLevels, level) {
			m.errorMsg = fmt.Sprintf("Valid levels: %s", strings.Join(validLogLevels, "/"))
			return m, nil
		}
		m.config.LogLevel = level
		m.state = StateConfigLogFormat
		m.errorMsg = ""

	case StateConfigLogFormat:
		format := strings.ToLower(strings.TrimSpace(m.input))
		if !contains(validLogFormats, format) {
			m.errorMsg = fmt.Sprintf("Valid formats: %s", strings.Join(validLogFormats, "/"))
			return m, nil
		}
		m.config.LogFormat = format
		m.state = StatePathConfig
		m.errorMsg = ""

	case StatePathConfig:
		m.state = StateConfigDir
		m.input, _ = os.Getwd()
		m.errorMsg = ""

	case StateConfigDir:
		configDir := strings.TrimSpace(m.input)
		if configDir == "" {
			configDir, _ = os.Getwd()
		}
		m.config.ConfigPath = filepath.Join(configDir, "config.yaml")
		m.state = StateSystemdBinaryPath
		m.errorMsg = ""

	case StateSystemdBinaryPath:
		m.state = StateEndpointHost
		m.endpoints = make([]endpointConfig, 0)
		m.currentEndpoint = endpointConfig{}
		m.errorMsg = ""

	case StateEndpointHost:
		host := strings.TrimSpace(m.input)
		if host == "" {
			if len(m.endpoints) > 0 {
				m.config.Endpoints = m.endpoints
				m.state = StateSystemdService
			} else {
				m.errorMsg = "At least one endpoint is required"
			}
		} else {
			m.currentEndpoint = endpointConfig{Host: host}
			m.state = StateModelID
			m.models = make([]modelConfig, 0)
			m.errorMsg = ""
		}

	case StateModelID:
		modelID := strings.TrimSpace(m.input)
		if modelID == "" {
			if len(m.models) > 0 {
				m.currentEndpoint.Models = m.models
				m.endpoints = append(m.endpoints, m.currentEndpoint)
				m.currentEndpoint = endpointConfig{}
				m.state = StateEndpointHost
				m.input = ""
			} else {
				m.errorMsg = "At least one model is required"
			}
		} else {
			m.currentModel = modelConfig{ID: modelID}
			m.state = StateModelPath
			m.errorMsg = ""
		}

	case StateModelPath:
		path := strings.TrimSpace(m.input)
		if path == "" {
			path = "/v1/chat/completions"
		}
		if !strings.HasPrefix(path, "/") {
			path = "/" + path
		}
		m.currentModel.Path = path
		m.state = StateModelName
		m.input = ""

	case StateModelName:
		name := strings.TrimSpace(m.input)
		if name == "" {
			name = m.currentModel.ID
		}
		m.currentModel.ModelName = name
		m.state = StateModelTemp
		m.input = "0.7"

	case StateModelTemp:
		temp := parseFloat(m.input, 0.7)
		m.currentModel.Temperature = temp
		m.state = StateModelTopP
		m.input = "0.95"

	case StateModelTopP:
		topP := parseFloat(m.input, 0.95)
		m.currentModel.TopP = topP
		m.state = StateModelTopK
		m.input = "20"

	case StateModelTopK:
		topK := parseInt(m.input, 20)
		m.currentModel.TopK = topK
		m.state = StateModelStream
		m.input = "y"

	case StateModelStream:
		input := strings.ToLower(strings.TrimSpace(m.input))
		m.currentModel.Stream = input == "y" || input == "yes"
		m.state = StateModelRepPenalty
		m.input = "1.0"

	case StateModelRepPenalty:
		penalty := parseFloat(m.input, 1.0)
		m.currentModel.RepetitionPenalty = penalty
		m.models = append(m.models, m.currentModel)
		m.state = StateModelID
		m.input = ""
		m.errorMsg = ""

	case StateSystemdService:
		input := strings.ToLower(strings.TrimSpace(m.input))
		m.config.SystemdService = input == "y" || input == "yes"
		if m.config.SystemdService {
			m.state = StateServiceUser
			m.input = "nobody"
		} else {
			m.state = StateSummary
		}
		m.errorMsg = ""

	case StateServiceUser:
		user := strings.TrimSpace(m.input)
		if user == "" {
			user = "nobody"
		}
		m.config.ServiceUser = user
		m.state = StateSummary
		m.errorMsg = ""

	case StateSummary:
		input := strings.ToLower(strings.TrimSpace(m.input))
		if input == "y" || input == "yes" {
			m.state = StateGenerating
			m.errorMsg = ""
			return m, tea.Quit
		}
		m.state = StateError
		m.errorMsg = "Installation cancelled"

	case StateError:
		return m, tea.Quit
	}

	return m, nil
}

func (m model) handleBack() (model, tea.Cmd) {
	switch m.state {
	case StateConfigPort:
		m.state = StateServerConfig
	case StateConfigLogLevel:
		m.state = StateConfigPort
	case StateConfigLogFormat:
		m.state = StateConfigLogLevel
	case StateConfigDir:
		m.state = StatePathConfig
	case StateEndpointHost:
		if len(m.endpoints) > 0 {
			lastEp := m.endpoints[len(m.endpoints)-1]
			m.currentEndpoint = lastEp
			m.endpoints = m.endpoints[:len(m.endpoints)-1]
			m.models = lastEp.Models
			if len(m.models) > 0 {
				m.currentModel = m.models[len(m.models)-1]
				m.models = m.models[:len(m.models)-1]
				m.state = StateModelID
			} else {
				m.state = StateModelID
			}
		} else {
			m.state = StateSystemdBinaryPath
		}
	case StateServiceUser:
		m.state = StateSystemdService
	case StateSummary:
		m.state = m.previousStep()
	}
	m.errorMsg = ""
	return m, nil
}

func (m model) previousStep() State {
	switch m.state {
	case StateSummary:
		return StateSystemdService
	}
	return StateServerConfig
}

// View renders the model
func (m model) View() string {
	var content strings.Builder

	switch m.state {
	case StateStart:
		content.WriteString(m.viewStart())

	case StateServerConfig:
		content.WriteString(m.viewServerConfig())

	case StateConfigPort, StateConfigLogLevel, StateConfigLogFormat:
		content.WriteString(m.viewConfigField())

	case StatePathConfig:
		content.WriteString(m.viewPathConfig())

	case StateConfigDir:
		content.WriteString(m.viewConfigDir())

	case StateSystemdBinaryPath:
		content.WriteString(m.viewSystemdBinaryPath())

	case StateEndpointHost, StateModelID, StateModelPath, StateModelName,
	     StateModelTemp, StateModelTopP, StateModelTopK, StateModelStream,
	     StateModelRepPenalty:
		content.WriteString(m.viewEndpointConfig())

	case StateSystemdService, StateServiceUser:
		content.WriteString(m.viewSystemdService())

	case StateSummary:
		content.WriteString(m.viewSummary())

	case StateError:
		content.WriteString(m.viewError())
	}

	if m.errorMsg != "" {
		content.WriteString("\n\n" + errorStyle.Render("⚠ "+m.errorMsg))
	}

	return content.String()
}

func (m model) viewStart() string {
	content := strings.Builder{}

	content.WriteString(titleStyle.Render("LM-Proxy Quick Install Wizard") + "\n")
	content.WriteString(strings.Repeat("=", 40) + "\n\n")
	content.WriteString(infoStyle.Render("This wizard will guide you through configuring LM-Proxy.") + "\n\n")
	content.WriteString(hintStyle.Render("Press Enter to continue, or Ctrl+C to quit"))

	return content.String()
}

func (m model) viewServerConfig() string {
	content := strings.Builder{}

	content.WriteString(headerStyle.Render("--- Server Configuration ---") + "\n\n")
	content.WriteString(labelStyle.Render("Listen Port:") + hintStyle.Render(fmt.Sprintf("[%d]", defaultPort)) + "\n")
	content.WriteString(labelStyle.Render("Log Level:") + hintStyle.Render(fmt.Sprintf("[%s] (%s)", defaultLogLevel, strings.Join(validLogLevels, "/"))) + "\n")
	content.WriteString(labelStyle.Render("Log Format:") + hintStyle.Render(fmt.Sprintf("[%s] (%s)", defaultLogFormat, strings.Join(validLogFormats, "/"))) + "\n\n")

	content.WriteString(hintStyle.Render("Press Enter to configure these values"))
	content.WriteString("\n" + hintStyle.Render("(Left/Ctrl+B to go back, Ctrl+C to quit)"))

	return content.String()
}

func (m model) viewConfigField() string {
	content := strings.Builder{}

	var prompt, hint string
	var defaultValue string

	switch m.state {
	case StateConfigPort:
		content.WriteString(headerStyle.Render("--- Server Configuration ---") + "\n\n")
		prompt = "Listen Port"
		hint = "1-65535"
		defaultValue = fmt.Sprintf("%d", defaultPort)
	case StateConfigLogLevel:
		content.WriteString(headerStyle.Render("--- Server Configuration ---") + "\n\n")
		prompt = "Log Level"
		hint = strings.Join(validLogLevels, "/")
		defaultValue = defaultLogLevel
	case StateConfigLogFormat:
		content.WriteString(headerStyle.Render("--- Server Configuration ---") + "\n\n")
		prompt = "Log Format"
		hint = strings.Join(validLogFormats, "/")
		defaultValue = defaultLogFormat
	}

	content.WriteString(labelStyle.Render(prompt+":") + " ")
	if m.input != "" {
		content.WriteString(m.input)
	} else {
		content.WriteString(hintStyle.Render(defaultValue))
	}
	if hint != "" {
		content.WriteString("\n" + hintStyle.Render(hint))
	}

	return content.String()
}

func (m model) viewPathConfig() string {
	content := strings.Builder{}

	content.WriteString(headerStyle.Render("--- Path Configuration ---") + "\n\n")
	content.WriteString(infoStyle.Render("The configuration file will be created in this directory.") + "\n\n")

	cwd, _ := os.Getwd()
	content.WriteString(labelStyle.Render("Install Directory:") + hintStyle.Render(fmt.Sprintf("[%s]", cwd)) + "\n\n")

	content.WriteString(hintStyle.Render("Press Enter to continue"))
	content.WriteString("\n" + hintStyle.Render("(Left/Ctrl+B to go back, Ctrl+C to quit)"))

	return content.String()
}

func (m model) viewConfigDir() string {
	content := strings.Builder{}

	content.WriteString(headerStyle.Render("--- Path Configuration ---") + "\n\n")
	content.WriteString(labelStyle.Render("Install Directory:") + " ")
	if m.input != "" {
		content.WriteString(m.input)
	} else {
		cwd, _ := os.Getwd()
		content.WriteString(hintStyle.Render(cwd))
	}

	return content.String()
}

func (m model) viewSystemdBinaryPath() string {
	content := strings.Builder{}

	content.WriteString(headerStyle.Render("--- Binary Installation ---") + "\n\n")
	content.WriteString(infoStyle.Render("The binary will be installed to: "+m.config.ConfigPath) + "\n\n")
	content.WriteString(infoStyle.Render("Answer 'yes' to install to /usr/local/bin (requires sudo)") + "\n")
	content.WriteString(infoStyle.Render("Answer 'no' to install in the selected directory") + "\n\n")

	content.WriteString(labelStyle.Render("Install to /usr/local/bin?") + " [y/N]")

	return content.String()
}

func (m model) viewEndpointConfig() string {
	content := strings.Builder()

	content.WriteString(headerStyle.Render("--- Endpoint Configuration ---") + "\n\n")

	if m.state == StateEndpointHost {
		if len(m.endpoints) == 0 {
			content.WriteString(infoStyle.Render("You need at least one LLM endpoint to configure.") + "\n\n")
			content.WriteString(labelStyle.Render("LLM Server URL:") + " ")
		} else {
			content.WriteString(infoStyle.Render(fmt.Sprintf("Enter another endpoint (or leave empty to finish)") + "\n\n"))
			content.WriteString(labelStyle.Render("LLM Server URL:") + " ")
		}
	} else {
		// Model configuration
		epIndex := len(m.endpoints)
		modelIndex := len(m.models) + 1

		content.WriteString(infoStyle.Render(fmt.Sprintf("Endpoint %d: %s", epIndex+1, m.currentEndpoint.Host)) + "\n\n")

		if m.state == StateModelID {
			if len(m.models) == 0 {
				content.WriteString(infoStyle.Render("Add at least one model to this endpoint.") + "\n\n")
			} else {
				content.WriteString(infoStyle.Render("Add another model (or leave empty to finish)") + "\n\n")
			}
			content.WriteString(labelStyle.Render(fmt.Sprintf("Model ID (%d/?)", modelIndex)) + " ")
		} else {
			content.WriteString(labelStyle.Render("Model ID:") + " " + m.currentModel.ID + "\n\n")

			switch m.state {
			case StateModelPath:
				content.WriteString(labelStyle.Render("Proxy Path:") + " ")
				defaultPath := "/v1/chat/completions"
				if m.input != "" {
					content.WriteString(m.input)
				} else {
					content.WriteString(hintStyle.Render(defaultPath))
				}
			case StateModelName:
				content.WriteString(labelStyle.Render("Backend Model Name:") + " ")
				if m.input != "" {
					content.WriteString(m.input)
				} else {
					content.WriteString(hintStyle.Render("(default: "+m.currentModel.ID+")"))
				}
			case StateModelTemp:
				content.WriteString(labelStyle.Render("Temperature:") + " ")
				if m.input != "" {
					content.WriteString(m.input)
				} else {
					content.WriteString(hintStyle.Render("(default: 0.7)"))
				}
				content.WriteString(" [0.0-2.0]")
			case StateModelTopP:
				content.WriteString(labelStyle.Render("Top P:") + " ")
				if m.input != "" {
					content.WriteString(m.input)
				} else {
					content.WriteString(hintStyle.Render("(default: 0.95)"))
				}
				content.WriteString(" [0.0-1.0]")
			case StateModelTopK:
				content.WriteString(labelStyle.Render("Top K:") + " ")
				if m.input != "" {
					content.WriteString(m.input)
				} else {
					content.WriteString(hintStyle.Render("(default: 20)"))
				}
			case StateModelStream:
				content.WriteString(labelStyle.Render("Enable Streaming:") + " [Y/n]")
			case StateModelRepPenalty:
				content.WriteString(labelStyle.Render("Repetition Penalty:") + " ")
				if m.input != "" {
					content.WriteString(m.input)
				} else {
					content.WriteString(hintStyle.Render("(default: 1.0)"))
				}
			}
		}
	}

	return content.String()
}

func (m model) viewSystemdService() string {
	content := strings.Builder{}

	if m.state == StateSystemdService {
		content.WriteString(headerStyle.Render("--- Systemd Service ---") + "\n\n")
		content.WriteString(infoStyle.Render("Generate a systemd service file for easy management.") + "\n\n")
		content.WriteString(labelStyle.Render("Generate systemd service?") + " [Y/n]")
	} else {
		content.WriteString(headerStyle.Render("--- Systemd Service ---") + "\n\n")
		content.WriteString(labelStyle.Render("Service User:") + " ")
		if m.input != "" {
			content.WriteString(m.input)
		} else {
			content.WriteString("nobody")
		}
	}

	return content.String()
}

func (m model) viewSummary() string {
	content := strings.Builder{}

	content.WriteString(headerStyle.Render("Configuration Summary") + "\n")
	content.WriteString(strings.Repeat("-", 24) + "\n\n")

	content.WriteString(boxStyle.Render(
		labelStyle.Render("Server Port:")+fmt.Sprintf("%d\n", m.config.ServerPort)+
			labelStyle.Render("Log Level:")+" "+m.config.LogLevel+"\n"+
			labelStyle.Render("Log Format:")+" "+m.config.LogFormat+"\n"+
			labelStyle.Render("Config File:")+m.config.ConfigPath+"\n"+
			labelStyle.Render("Endpoints:")+fmt.Sprintf(" %d\n", len(m.config.Endpoints)),
	) + "\n")

	for i, ep := range m.config.Endpoints {
		for j, model := range ep.Models {
			epNum := i + 1
			modelNum := j + 1
			if j == 0 {
				content.WriteString(infoStyle.Render(fmt.Sprintf("  Endpoint %d: %s (%d model(s))\n", epNum, ep.Host, len(ep.Models))))
			}
			content.WriteString(fmt.Sprintf("    %d.%d %s\n", epNum, modelNum, model.ID))
		}
	}

	if m.config.SystemdService {
		content.WriteString("\n" + boxStyle.Render(
			labelStyle.Render("Systemd Service:")+(" enabled\n")+
				labelStyle.Render("Service User:")+" "+m.config.ServiceUser,
		) + "\n")
	}

	content.WriteString("\n" + labelStyle.Render("Generate configuration files?") + " [Y/n]")

	return content.String()
}

func (m model) viewError() string {
	content := strings.Builder()

	if m.errorMsg != "" {
		content.WriteString(errorStyle.Render(m.errorMsg))
	} else {
		content.WriteString(successStyle.Render("✅ Setup complete!"))
	}

	return content.String()
}

// Helper functions
func parseInt(s string, defaultValue int) int {
	var result int
	if _, err := fmt.Sscanf(s, "%d", &result); err != nil {
		return defaultValue
	}
	return result
}

func parseFloat(s string, defaultValue float64) float64 {
	var result float64
	if _, err := fmt.Sscanf(s, "%f", &result); err != nil {
		return defaultValue
	}
	return result
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// Finalize creates the configuration files
func (m model) Finalize() error {
	if m.state != StateGenerating {
		return fmt.Errorf("not in generating state")
	}

	// Create config.yaml
	if err := generateConfigFile(m.config); err != nil {
		return fmt.Errorf("failed to generate config.yaml: %w", err)
	}

	fmt.Println("\n" + successStyle.Render("✓ Generated: ") + m.config.ConfigPath)

	// Generate systemd service
	if m.config.SystemdService {
		servicePath := filepath.Join(filepath.Dir(m.config.ConfigPath), "lmproxy.service")
		if err := generateSystemdService(m.config, m.config.ServiceUser, servicePath); err != nil {
			return fmt.Errorf("failed to generate systemd service: %w", err)
		}
		fmt.Println(successStyle.Render("✓ Generated: ") + servicePath)
	}

	// Print next steps
	printNextSteps(m.config)

	return nil
}

func printNextSteps(cfg installConfig) {
	fmt.Println()
	fmt.Println(successStyle.Render("✅ Installation complete!"))
	fmt.Println()
	fmt.Println(headerStyle.Render("Next steps:"))
	fmt.Println("1. Review and edit config.yaml if needed")
	fmt.Println("2. Build the proxy: go build -o lmproxy main.go")
	fmt.Println("3. Run the proxy: ./lmproxy config.yaml")

	if cfg.SystemdService {
		fmt.Println("\nTo install as a systemd service:")
		fmt.Println("4. Install systemd service: sudo install -m 644 lmproxy.service /etc/systemd/system/")
		fmt.Println("5. Enable and start: sudo systemctl daemon-reload && sudo systemctl enable --now lmproxy")
		fmt.Println("6. Check status: sudo systemctl status lmproxy")
		fmt.Println("7. View logs: sudo journalctl -u lmproxy -f")
	}
}

func generateConfigFile(cfg installConfig) error {
	if err := os.MkdirAll(filepath.Dir(cfg.ConfigPath), 0755); err != nil {
		return err
	}

	content := fmt.Sprintf(`# LM-Proxy Configuration
# Generated by installer wizard

server:
  host: %s
  port: %d
  max_request_body_size: %d
  timeout: %d

logging:
  level: %s
  format: %s

endpoints:
`, cfg.ServerHost, cfg.ServerPort, cfg.MaxRequestBodySize, cfg.Timeout, cfg.LogLevel, cfg.LogFormat)

	for _, ep := range cfg.Endpoints {
		content += fmt.Sprintf("  - host: %s\n    models:\n", ep.Host)
		for _, model := range ep.Models {
			content += fmt.Sprintf(`      - id: %s
        path: %s
        body:
          model: %s
          temperature: %.2f
          top_p: %.2f
          top_k: %d
          stream: %t
          repetition_penalty: %.2f
`, model.ID, model.Path, model.ModelName, model.Temperature, model.TopP, model.TopK, model.Stream, model.RepetitionPenalty)
		}
	}

	return os.WriteFile(cfg.ConfigPath, []byte(content), 0644)
}

func generateSystemdService(cfg installConfig, serviceUser, path string) error {
	content := fmt.Sprintf(`[Unit]
Description=LM-Proxy Service
After=network.target

[Service]
Type=simple
User=%s
ExecStart=%s %s
Restart=on-failure
RestartSec=5

# Security hardening
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=%s

[Install]
WantedBy=multi-user.target
`, serviceUser, cfg.BinaryPath, cfg.ConfigPath, cfg.AltRootDir)

	return os.WriteFile(path, []byte(content), 0644)
}

func main() {
	initialModel := model{
		state: StateStart,
	}

	p := tea.NewProgram(
		initialModel,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	finalModel, err := p.Run()
	if err != nil {
		fmt.Printf("Error starting wizard: %v\n", err)
		os.Exit(1)
	}

	m := finalModel.(model)
	if m.state == StateError && m.errorMsg == "Installation cancelled" {
		fmt.Println("\nInstallation cancelled.")
		os.Exit(0)
	}

	if m.state == StateGenerating || m.state == StateComplete {
		if err := m.Finalize(); err != nil {
			fmt.Printf("\nError generating files: %v\n", err)
			os.Exit(1)
		}
	}
}
