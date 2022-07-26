package main

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	_ "github.com/lib/pq"
)

type model struct {
	inputs     []textinput.Model
	focused    int
	loading    bool
	submitting bool
	typing     bool
	spinner    spinner.Model
	err        error
	db         *sql.DB
}

func main() {
	m := initialModel()
	p := tea.NewProgram(m)

	m.inputs[wanIp].Validate = m.validateWanIP

	if err := p.Start(); err != nil {
		log.Fatal(err)
	}
}

const (
	username = iota
	wanIp
	speed
	password
)

func initialModel() model {
	var inputs []textinput.Model = make([]textinput.Model, 4)

	inputs[username] = textinput.New()
	inputs[username].Focus()
	inputs[username].CharLimit = 30
	inputs[username].Placeholder = "Username..."
	inputs[username].Width = 30
	inputs[username].Prompt = ""
	inputs[username].Validate = validateUsername

	inputs[wanIp] = textinput.New()
	inputs[wanIp].CharLimit = 16
	inputs[wanIp].Placeholder = "Wan Ip..."
	inputs[wanIp].Width = 30
	inputs[wanIp].Prompt = ""

	inputs[speed] = textinput.New()
	inputs[speed].CharLimit = 6
	inputs[speed].Placeholder = "Speed (100M)"
	inputs[speed].Width = 30
	inputs[speed].Prompt = ""
	inputs[speed].Validate = validateSpeed

	inputs[password] = textinput.New()
	inputs[password].CharLimit = 30
	inputs[password].Placeholder = "Password..."
	inputs[password].Width = 30
	inputs[password].Prompt = ""
	inputs[password].Validate = validatePassword

	s := spinner.New()
	s.Spinner = spinner.Dot
	const (
		host     = "localhost"
		port     = 5432
		user     = "finn"
		password = "1234"
		dbname   = "tuitest"
	)
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)
	// connstr := "postgresql://finn:123@127.0.0.1/tuitest?sslmode=disable"

	db, err := sql.Open("postgres", psqlInfo)

	if err != nil {
		log.Fatal(err)
	}

	return model{
		inputs:     inputs,
		focused:    0,
		err:        nil,
		typing:     true,
		loading:    false,
		submitting: false,
		spinner:    s,
		db:         db,
	}
}
func containsTimes(s []string, check string) int {
	count := 0
	for i := range s {
		if s[i] == check {
			count++
		}
	}
	return count
}
func (m model) validateWanIP(s string) error {

	split := strings.Split(s, ".")

	if containsTimes(split, "") > 1 {
		return fmt.Errorf("nothing between decimals")
	}
	if len(split) > 4 {
		return fmt.Errorf("too many octets")
	}
	for i := range split {

		if split[i] != "" {
			if len(split[i]) > 3 {
				m.err = fmt.Errorf("%s is not a valid octet", split[i])
				return m.err
			}
			if no, err := strconv.Atoi(split[i]); err != nil || no > 255 {
				m.err = fmt.Errorf("%s is not a valid octet", split[i])
				return m.err
			}
		}

	}
	return nil
}

func validateUsername(s string) error {
	split := strings.Split(s, " ")
	if len(split) > 1 {
		return fmt.Errorf("No spaces in username")
	}
	return nil
}

func validatePassword(s string) error {
	split := strings.Split(s, " ")
	if len(split) > 1 {
		return fmt.Errorf("No spaces in password")
	}
	return nil
}

func validateSpeed(s string) error {
	split := strings.Split(s, " ")
	if len(split) > 1 {
		return fmt.Errorf("No spaces in Speed")
	}
	return nil
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {

	switch msg := msg.(type) {
	case dbreturn:
		if err := msg.err; err != nil {
			m.err = err
			return m, nil
		}
		m.typing = false
		m.loading = false
		m.submitting = false
		return m, nil

	case validateError:
		m.typing = true
		m.focused = msg.id
		m.inputs[msg.id].Err = msg.err
		m.loading = false
		m.submitting = false
	case tea.KeyMsg:
		// fmt.Printf(msg.String())
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "tab":
			m.focused = (m.focused + 1) % len(m.inputs)

		case "enter":
			if m.typing {
				m.submitting = true
				m.typing = false
				for i := range m.inputs {
					m.inputs[i].Blur()
				}
			}
			if !m.typing && !m.loading && !m.submitting {
				m.typing = true
				for i := range m.inputs {
					m.inputs[i].Reset()
					m.inputs[i].Blur()
				}
				m.focused = 0
				m.inputs[0].Focus()
				return m, nil
			}
		case "y":
			if m.submitting {
				m.submitting = false
				m.typing = false
				m.loading = true
				return m, tea.Batch(spinner.Tick, m.addToDatabase())
			}
		case "n":
			if m.submitting {
				m.submitting = false
				m.loading = false
				m.typing = true

			}
		}
		if m.typing {
			for i := range m.inputs {
				m.inputs[i].Blur()
				m.inputs[i].Err = nil
			}
			m.inputs[m.focused].Focus()
		}

	}

	if m.typing {
		var (
			cmds []tea.Cmd = make([]tea.Cmd, len(m.inputs))
		)
		for i := range m.inputs {
			m.inputs[i], cmds[i] = m.inputs[i].Update(msg)
		}
		return m, tea.Batch(cmds...)
	}
	return m, nil
}

func (m model) View() string {
	if err := m.err; err != nil {
		return fmt.Sprintf("Received Error: %v", err)
	}
	if m.submitting {
		return fmt.Sprint("Press y/n to continue or go back")
	}
	if m.typing {
		view := fmt.Sprintf(`
		%s %s  %s
		
		%s %s  %s
	
		%s %s  %s
	
		%s %s  %s

			Enter to continue
		`, "Username: ", m.inputs[username].View(), errToString(m.inputs[username].Err), "Wan IP: ", m.inputs[wanIp].View(), errToString(m.inputs[wanIp].Err), "Speed: ", m.inputs[speed].View(), errToString(m.inputs[speed].Err), "Password: ", m.inputs[password].View(), errToString(m.inputs[password].Err))
		return view
	}
	if m.loading && !m.typing {
		return fmt.Sprintf("Loading...  %s", m.spinner.View())
	}

	if m.err == nil && !m.typing && !m.loading && !m.submitting {
		return fmt.Sprintf("Added %s to RADIUS       ctrl+c to exit or enter to create another", m.inputs[username].Value())
	}
	return ""

}

type dbreturn struct {
	err     error
	success bool
}

type validateError struct {
	err error
	id  int
}

func (m model) addToDatabase() tea.Cmd {
	if user := m.inputs[username].Value(); strings.TrimSpace(user) == "" {
		return func() tea.Msg {
			return validateError{
				err: fmt.Errorf("username cannot be null"),
				id:  username,
			}
		}
	}
	if _wanIp := m.inputs[wanIp].Value(); strings.TrimSpace(_wanIp) == "" {
		return func() tea.Msg {
			return validateError{
				err: fmt.Errorf("wanIp cannot be null"),
				id:  wanIp,
			}
		}
	}
	if _wanIp := m.inputs[wanIp].Value(); strings.Index((strings.Split(_wanIp, ".")[0]), "0") == 0 {
		return func() tea.Msg {
			return validateError{
				err: fmt.Errorf("invalid wan ip"),
				id:  wanIp,
			}
		}
	}
	if _speed := m.inputs[speed].Value(); strings.TrimSpace(_speed) == "" {
		return func() tea.Msg {
			return validateError{
				err: fmt.Errorf("speed cannot be null"),
				id:  speed,
			}
		}
	}
	if _password := m.inputs[password].Value(); strings.TrimSpace(_password) == "" {
		return func() tea.Msg {
			return validateError{
				err: fmt.Errorf("password cannot be null"),
				id:  password,
			}
		}
	}
	return func() tea.Msg {
		_, err := m.db.Exec("Insert into usergroup Values ($1,$2,$3,$4)", m.inputs[username].Value(), m.inputs[wanIp].Value(), m.inputs[speed].Value(), m.inputs[password].Value())
		if err != nil {
			return dbreturn{
				success: false,
				err:     err,
			}
		}
		return dbreturn{
			success: true,
			err:     nil,
		}
	}

}

func errToString(e error) string {
	if e == nil {
		return ""
	}
	return fmt.Sprintf("%v", e)
}
