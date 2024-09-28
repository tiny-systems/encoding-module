package encode

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"github.com/tiny-systems/module/module"
	"github.com/tiny-systems/module/registry"
)

const (
	ComponentName = "xml_encode"
	RequestPort   = "request"
	ResponsePort  = "response"
	ErrorPort     = "error"
)

type Context any

type Settings struct {
	EnableErrorPort bool `json:"enableErrorPort" required:"true" title:"Enable Error Port" description:"If error happen, error port will emit an error message"`
}

type Error struct {
	Context Context `json:"context"`
	Error   string  `json:"error"`
}

type Request struct {
	Context  Context `json:"context" configurable:"true" title:"Context" description:"Arbitrary message to be send alongside with encoded message"`
	Document any     `json:"document" required:"true" configurable:"true" title:"Input object" description:""`
}

type Response struct {
	Context Context `json:"context"`
	Encoded string  `json:"encoded"`
}

type Component struct {
	settings Settings
}

func (h *Component) GetInfo() module.ComponentInfo {
	return module.ComponentInfo{
		Name:        ComponentName,
		Description: "XML Encoder",
		Info:        "Encodes input document with XML encoded data",
		Tags:        []string{"xml"},
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

		b := bytes.NewBuffer(nil)
		enc := xml.NewEncoder(b)

		err := enc.Encode(in.Document)
		if err != nil {
			if !h.settings.EnableErrorPort {
				return err
			}
			return handler(ctx, ErrorPort, Error{
				Context: in.Context,
				Error:   err.Error(),
			})
		}

		return handler(ctx, ResponsePort, Response{
			Encoded: b.String(),
			Context: in.Context,
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
			Source:        true,
			Configuration: Request{},
		},
		{
			Name:          ResponsePort,
			Position:      module.Right,
			Label:         "Out",
			Configuration: Response{},
		},
		{
			Name:          module.SettingsPort,
			Label:         "Settings",
			Source:        true,
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
		Source:        false,
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
