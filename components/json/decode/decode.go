package decode

import (
	"context"
	"fmt"
	"github.com/goccy/go-json"
	"github.com/tiny-systems/module/module"
	"github.com/tiny-systems/module/registry"
)

const (
	ComponentName = "json_decode"
	RequestPort   = "request"
	ResponsePort  = "response"
	ErrorPort     = "error"
)

type Context any

type Decoded any

type Settings struct {
	EnableErrorPort bool    `json:"enableErrorPort" required:"true" title:"Enable Error Port" description:"If error happen, error port will emit an error message"`
	Decoded         Decoded `json:"decoded" configurable:"true" title:"Decoded Document Example" description:"Define document schema. Optional."`
}

type Error struct {
	Context Context `json:"context"`
	Error   string  `json:"error"`
}

type Request struct {
	Context Context `json:"context,omitempty" configurable:"true" title:"Context" description:"Arbitrary message to be send alongside with decoded message"`
	Encoded string  `json:"encoded" required:"true" title:"Input string" format:"textarea" description:"JSON encoded string"`
}

type Output struct {
	Context Context `json:"context"`
	Decoded Decoded `json:"decoded" configurable:"true"`
}

type Component struct {
	settings Settings
}

func (h *Component) GetInfo() module.ComponentInfo {
	return module.ComponentInfo{
		Name:        ComponentName,
		Description: "JSON Decoder",
		Info:        "Decodes input string with JSON",
		Tags:        []string{"json"},
	}
}

func (h *Component) Handle(ctx context.Context, handler module.Handler, port string, msg interface{}) error {

	switch port {
	case module.SettingsPort:

		in, ok := msg.(Settings)
		if !ok {
			return fmt.Errorf("invalid settings")
		}
		h.settings = in

	case RequestPort:

		in, ok := msg.(Request)
		if !ok {
			return fmt.Errorf("invalid input")
		}

		var res Decoded

		err := json.Unmarshal([]byte(in.Encoded), &res)
		if err != nil {
			if !h.settings.EnableErrorPort {
				return err
			}
			return handler(ctx, ErrorPort, Error{
				Context: in.Context,
				Error:   err.Error(),
			})
		}

		return handler(ctx, ResponsePort, Output{
			Context: in.Context,
			Decoded: res,
		})

	default:
		return fmt.Errorf("port %s is not supoprted", port)
	}
	return nil
}

func (h *Component) Ports() []module.Port {
	ports := []module.Port{
		{
			Name:          RequestPort,
			Label:         "In",
			Position:      module.Left,
			Configuration: Request{},
		},
		{
			Name:          ResponsePort,
			Position:      module.Right,
			Label:         "Out",
			Source:        true,
			Configuration: Output{},
		},
		{
			Name:          module.SettingsPort,
			Label:         "Settings",
			Configuration: h.settings,
		},
	}
	if !h.settings.EnableErrorPort {
		return ports
	}
	return append(ports, module.Port{
		Position:      module.Bottom,
		Name:          ErrorPort,
		Label:         "Error",
		Source:        true,
		Configuration: Error{},
	})
}

func (h *Component) Instance() module.Component {
	return &Component{
		settings: Settings{},
	}
}

var _ module.Component = (*Component)(nil)

func init() {
	registry.Register(&Component{})
}
