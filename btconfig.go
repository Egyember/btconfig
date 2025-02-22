package main

import (
	"errors"
	"fmt"
	"math"
	"slices"
	"strings"

	"btconfig/ansi"

	tea "github.com/charmbracelet/bubbletea"
	bt "tinygo.org/x/bluetooth"
)

var ErrWriteLess = errors.New("wrote less then specified string")

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

type conncetion struct {
	connected       bt.Device
	services        []bt.DeviceService
	characteristics []bt.DeviceCharacteristic
}

type wificonf struct {
	passwd, ssid  *string
	channel, auth uint8
}
type model struct {
	presses []string
	scan    bool
	term    struct {
		x, y int
	}
	adapter         *bt.Adapter
	btdevs          *btdevs
	cursor          int
	scanCursorMax   int
	configCursormax int
	selected        int
	conncetion      *conncetion
	textInPut       bool
	text            **string
	wificonfig      *wificonf
	err             error
}

func initialModel() model {
	m := model{}
	m.term.x = 0
	m.term.y = 0
	m.scan = false
	m.adapter = bt.DefaultAdapter
	m.adapter.Enable()
	m.scanCursorMax = 0
	m.configCursormax = 4
	m.cursor = 0
	m.selected = -1
	m.conncetion = nil
	m.btdevs = new(btdevs)
	m.btdevs.add = make(chan btdev, 10)
	m.btdevs.scanResults = make([]btdev, 0)
	go m.btdevs.addResult()
	m.text = nil
	m.textInPut = false
	var strings [2]string
	m.wificonfig = new(wificonf)
	m.wificonfig.ssid = &strings[0]
	m.wificonfig.passwd = &strings[1]
	return m
}

func (s model) Init() tea.Cmd {
	return tea.ClearScreen
}

func (s model) parseKey(msg tea.KeyMsg) (model, tea.Cmd) {
	if s.adapter == nil {
		return s, nil
	}
	switch msg.String() {
	case "ctrl+c":
		if s.conncetion != nil {
			s.conncetion.connected.Disconnect()
		}
		return s, tea.Quit
	case "esc", "q":
		if s.selected == -1 {
			if s.conncetion != nil {
				s.conncetion.connected.Disconnect()
			}
			return s, tea.Quit
		} else {
			s.conncetion.connected.Disconnect()
			s.conncetion = nil
			s.selected = -1
		}
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
	case "up":
		if s.cursor > 0 {
			s.cursor--
		}
	case "down":
		if s.selected == -1 {
			if s.cursor < s.scanCursorMax {
				s.cursor++
			}
		} else {
			if s.cursor < s.configCursormax {
				s.cursor++
			}
		}
	case "enter":
		if s.err != nil {
			s.err = nil
			break
		}
		if s.selected == -1 {
			s.selected = s.cursor
			s.conncetion = new(conncetion)
			var err error
			s.conncetion.connected, err = s.adapter.Connect(s.btdevs.scanResults[s.selected].addr, bt.ConnectionParams{})
			if err != nil {
				s.err = err
				s.selected = -1
				break
			}

			serviceUUID := bt.NewUUID([16]byte{246, 35, 207, 46, 213, 119, 141, 146, 175, 79, 198, 129, 199, 180, 108, 235})
			s.conncetion.services, err = s.conncetion.connected.DiscoverServices([]bt.UUID{serviceUUID})
			if len(s.conncetion.services) != 1 && err == nil {
				err = errors.New("worng number of services returned")
			}
			if err != nil {
				s.conncetion.connected.Disconnect()
				s.err = err
				s.selected = -1
				break
			}
			s.conncetion.characteristics, err = s.conncetion.services[0].DiscoverCharacteristics(nil)
			if err != nil {
				s.conncetion.connected.Disconnect()
				s.err = err
				s.selected = -1
				break
			}

		} else {
			switch s.selected {
			case 0:
				s.text = &s.wificonfig.ssid
				s.textInPut = true
			case 1:
				s.text = &s.wificonfig.passwd
				s.textInPut = true
			case 4:
				s.sendWifi()
			}
		}
	case "r":
		return s, tea.ClearScreen
	default:
		s.presses = append(s.presses, msg.String())
	}
	return s, nil
}

func (s model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		s.term.x = msg.Width
		s.term.y = msg.Height
	case tea.KeyMsg:
		if s.textInPut {
			if s.text == nil {
				panic(errors.New("invalid text buffer"))
			}
			input := msg.String()
			if input == "enter" {
				s.textInPut = false
				s.text = nil
				break
			}
			output := new(string)
			*output = **s.text + input
			*s.text = output
		} else {
			return s.parseKey(msg)
		}
	case MsgNewDev:
		if s.selected == -1 {
			s.scanCursorMax = len(s.btdevs.scanResults) - 1
		}
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

func btsend(characteristic bt.DeviceCharacteristic, data []byte) error {
	n, err := characteristic.WriteWithoutResponse(data)
	if err != nil {
		return err
	}
	if n != len(data) {
		return ErrWriteLess
	}
	return nil
}

func (s model) sendWifi() error {
	var done bt.DeviceCharacteristic
	for _, v := range s.conncetion.characteristics {
		switch v.UUID() {
		case bt.NewUUID([16]byte{247, 35, 207, 46, 213, 119, 141, 146, 175, 79, 198, 129, 199, 180, 108, 235}): // ssid
			err := btsend(v, []byte(*s.wificonfig.ssid))
			if err != nil {
				return err
			}
		case bt.NewUUID([16]byte{247, 35, 207, 46, 213, 119, 141, 146, 175, 79, 198, 129, 199, 180, 108, 236}): // passwd
			err := btsend(v, []byte(*s.wificonfig.passwd))
			if err != nil {
				return err
			}
		case bt.NewUUID([16]byte{247, 35, 207, 46, 213, 119, 141, 146, 175, 79, 198, 129, 199, 180, 108, 237}): // channel
			err := btsend(v, []byte{s.wificonfig.channel})
			if err != nil {
				return err
			}
		case bt.NewUUID([16]byte{247, 35, 207, 46, 213, 119, 141, 146, 175, 79, 198, 129, 199, 180, 108, 238}): // AUTHMETOD
			err := btsend(v, []byte{s.wificonfig.auth})
			if err != nil {
				return err
			}
		case bt.NewUUID([16]byte{247, 35, 207, 46, 213, 119, 141, 146, 175, 79, 198, 129, 199, 180, 108, 239}): // done
			done = v
		}
	}
	readbuffer := make([]byte, 16)
	done.Read(readbuffer)
	return nil
}

func (s model) scanCallback(adapter *bt.Adapter, device bt.ScanResult) {
	// fmt.Println(device.LocalName())
	result := btdev{name: device.LocalName(), addr: device.Address}
	s.btdevs.add <- result
}

func (s model) View() string {
	st := s.RenderStatusBar()
	st += s.RenderMainContent()
	st += fmt.Sprintln(s.term.x, "x", s.term.y)
	st += fmt.Sprintln(s.cursor)
	st += "Button presses:\n"
	for _, v := range s.presses {
		st += v
	}
	if s.text != nil {
		st += fmt.Sprintf("*s.text: %p\n", *s.text)
		if *s.text != nil {
			st += fmt.Sprintf("**s.text: %p\n", **s.text)
		}

	}
	// padding to the end of the terminal
	numlines := strings.Count(st, "\n")
	padding := s.term.y - (numlines + 1)
	if padding > 0 {
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
	padding := s.term.x - (len(b) - ansi.CountAnsi(b))
	if padding > 0 {
		b += strings.Repeat(" ", padding)
	}
	b += "\n"
	return
}

func (s model) RenderMainContent() (b string) {
	if s.term.x == 0 {
		return ""
	}
	b = ""
	if s.err != nil {
		errTxt := s.err.Error()
		str, err := ansi.MidleText(ansi.SetColor(errTxt, ansi.BGred), s.term.x)
		if err != nil {
			panic(err)
		}
		b += str + "\n"
		b += "press enter to clear\n"
		return
	}
	if s.selected == -1 {
		data := make([][]string, len(s.btdevs.scanResults))
		for k, v := range s.btdevs.scanResults {
			data[k] = []string{v.name, v.addr.String()}
		}
		tableWith := []int{int(math.Floor(float64(s.term.x) / 2.0)), int(math.Ceil(float64(s.term.x) / 2.0))}
		b += ansi.Table([]string{"Name", "Addres"}, data, tableWith, s.cursor)
		return
	} else {
		data := [][]string{
			{"ssid", *s.wificonfig.ssid},
			{"password", *s.wificonfig.passwd},
			{"channel", string(s.wificonfig.channel)},
			{"auth", string(s.wificonfig.auth)},
		}
		tableWith := []int{int(math.Floor(float64(s.term.x) / 2.0)), int(math.Ceil(float64(s.term.x) / 2.0))}
		b += ansi.Table([]string{"Name", "value"}, data, tableWith, s.cursor)
		button, err := ansi.MidleText("send", s.term.x)
		if errors.Is(err, ansi.ErrTooLong) {
			if s.term.x >= 0 {
				text := "send"[:s.term.x-2]
				text += ">"
				button, _ = ansi.MidleText(text, s.term.x)
			}
		}
		if s.cursor == 4 {
			b += ansi.SetColor(button, ansi.BGblue)
			return
		}
		b += button
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
