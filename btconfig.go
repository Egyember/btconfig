package main

import (
	"fmt"
	"slices"
	"strings"

	"btconfig/ansi"

	tea "github.com/charmbracelet/bubbletea"
	bt "tinygo.org/x/bluetooth"
)

type btdev struct {
	name string
	addr bt.Address
}

type btdevs struct {
	scanResults []btdev
	add         chan btdev
	program     *tea.Program
}
type MsgNewDev struct{}

type model struct {
	presses []string
	scan    bool
	term    struct {
		x, y int
	}
	adapter  *bt.Adapter
	btdevs   *btdevs
	selected int
	err      error
}

func initialModel() model {
	m := model{}
	m.term.x = 0
	m.term.y = 0
	m.scan = false
	m.adapter = bt.DefaultAdapter
	m.adapter.Enable()
	m.selected = -1
	m.btdevs = new(btdevs)
	m.btdevs.add = make(chan btdev, 10)
	m.btdevs.scanResults = make([]btdev, 0)
	go m.btdevs.addResult()
	return m
}

func (s model) Init() tea.Cmd {
	return tea.ClearScreen
}

func (s model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		s.term.x = msg.Width
		s.term.y = msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return s, tea.Quit
		case "s":
			if s.scan {
				s.adapter.StopScan()
				s.scan = false
			} else {
				go func() {
					err := s.adapter.Scan(s.scanCallback)
					if err != nil {
						s.scan = false
						s.err = err
					}
				}()
				s.scan = true
			}
		default:
			s.presses = append(s.presses, msg.String())
		}
	case MsgNewDev:
		return s, nil
	}
	return s, nil
}

func (s *btdevs) addResult() {
	for i := range s.add {
		if !slices.Contains(s.scanResults, i) && i.name != "" {
			s.scanResults = append(s.scanResults, i)
			s.program.Send(MsgNewDev{})
		}
	}
}

func (s model) scanCallback(adapter *bt.Adapter, device bt.ScanResult) {
	// fmt.Println(device.LocalName())
	result := btdev{name: device.LocalName(), addr: device.Address}
	s.btdevs.add <- result
}

func (s model) View() string {
	st := s.RenderStatusBar()
	st += s.RenderMainContent()
	st += "Button presses:\n"
	for _, v := range s.presses {
		st += v
	}
	// padding to the end of the terminal
	numlines := strings.Count(st, "\n")
	padding := s.term.y - (numlines + 1)
	if s.term.y != 0 {
		st += strings.Repeat("\n", padding)
	}

	return st
}

func (s model) RenderStatusBar() (b string) {
	if s.term.x == 0 {
		return
	}
	if s.scan {
		b = ansi.SetColor("Scan on ", ansi.BGgreen)
	} else {
		b = ansi.SetColor("Scan off", ansi.BGred)
	}
	padding := s.term.x - len(b)
	b += strings.Repeat(" ", padding)
	b += "\n"
	return
}

func (s model) RenderMainContent() (b string) {
	b = ""
	if s.err != nil {
		errTxt := s.err.Error()
		str, err := ansi.MidleText(ansi.SetColor(errTxt, ansi.BGred), s.term.x)
		if err != nil {
			panic(err)
		}
		b += str
		return
	}
	for k, v := range s.btdevs.scanResults {
		b += fmt.Sprintf("%d: Name: %s, Addres: %s\n", k, v.name, v.addr.String())
	}
	return
}

func main() {
	model := initialModel()
	p := tea.NewProgram(model)
	model.btdevs.program = p
	_, err := p.Run()
	if err != nil {
		panic(err)
	}
}
