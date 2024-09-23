package gotemplate

import (
	"bytes"
	"context"
	"fmt"
	"github.com/tiny-systems/module/module"
	"github.com/tiny-systems/module/registry"
	"html/template"
	"time"
)

const (
	ComponentName = "go_template"
	RequestPort   = "request"
	ResponsePort  = "response"
	ErrorPort     = "error"
)

type Context any
type RenderData any

type Template struct {
	Name    string `json:"name" required:"true" title:"File name" Description:"e.g. footer.tmpl"`
	Content string `json:"content" required:"true" title:"Template" format:"textarea"`
}

type Settings struct {
	EnableErrorPort bool `json:"enableErrorPort" required:"true" title:"Enable Error Port" description:"If error happen, error port will emit an error message" tab:"Settings"`

	Templates []Template `json:"templates" required:"true" title:"Templates" minItems:"1" uniqueItems:"true" tab:"Templates"`
	Partials  []Template `json:"partials" required:"true" title:"Partials" description:"All partials being loaded with each template" minItems:"0" uniqueItems:"true" tab:"Partials"`
}

type Error struct {
	Context Context `json:"context"`
	Error   string  `json:"error"`
}

type Request struct {
	Context    Context    `json:"context,omitempty" configurable:"true" title:"Context" description:"Arbitrary message to be send alongside with rendered content"`
	RenderData RenderData `json:"renderData,omitempty" configurable:"true" title:"Render data" description:"Data being used to render the template"`
	Template   string     `json:"template" required:"true" title:"Template" description:"Template to render"`
}

type Response struct {
	Context Context `json:"context"`
	Content string  `json:"content"`
}

type Component struct {
	templateSet map[string]*template.Template
	settings    Settings
}

var defaultEngineSettings = Settings{
	Templates: []Template{
		{
			Name: "home.html",
			Content: `{{template "layout.html" .}}
{{define "title"}}Welcome.{{end}}
{{define "content"}}
Welcome
{{end}}`,
		},
		{
			Name: "page1.html",
			Content: `{{template "layout.html" .}}
{{define "title"}} Page one.{{end}}
{{define "content"}}
I'm page 1
{{end}}`,
		},
		{
			Name: "page2.html",
			Content: `{{template "layout.html" .}}
{{define "title"}} Page 2 title {{end}}
{{define "content"}}
I'm page 2
{{end}}`,
		},
	},
	Partials: []Template{
		{
			Name: "layout.html",
			Content: `<!DOCTYPE html>
<html lang="en">
<head>
<title>{{block "title" .}}{{end}}</title>
</head>
<body>
{{block "nav" . }}{{end}}
<div style="padding:20px">
{{block "content" .}}{{end}}
</div>
{{block "footer" .}}{{end}}
</body>
</html>`,
		},
		{
			Name: "footer.html",
			Content: `{{define "footer"}}
<hr/>
<div style="text-align:center">
 <p>&copy; {{now.UTC.Year}}</p>
 <p>{{builtWith}}</p>
</div>
{{end}}`,
		},
		{
			Name: "nav.html",
			Content: `{{define "nav"}}
<div>
 <a href="/">Home page</a>
 <a href="/page1">Page1</a>
 <a href="/page2">Page2</a>
</div>
{{end}}`,
		},
	},
}

func (h *Component) GetInfo() module.ComponentInfo {
	return module.ComponentInfo{
		Name:        ComponentName,
		Description: "Go Template Engine",
		Info:        "Renders templates using go's html/template standard package. Supports layouts and partials.",
		Tags:        []string{"html", "template", "engine"},
	}
}

func (h *Component) Handle(ctx context.Context, handler module.Handler, port string, msg interface{}) error {

	switch port {
	case module.SettingsPort:
		// compile template
		in, ok := msg.(Settings)
		if !ok {
			return fmt.Errorf("invalid settings")
		}

		h.settings = in
		ts := map[string]*template.Template{}

		funcMap := template.FuncMap{
			"now": time.Now,
			"builtWith": func() template.HTML {
				return `<a href="https://tinysystems.io?from=builtwith" target="_blank">Built with Tiny Systems</a>`
			},
		}

		for _, t := range in.Templates {
			tmpl, err := template.New(t.Name).Funcs(funcMap).Parse(t.Content)
			if err != nil {
				return err
			}
			for _, p := range in.Partials {
				_, err = tmpl.New(p.Name).Parse(p.Content)
				if err != nil {

					return err
				}
			}
			ts[t.Name] = tmpl
		}

		h.templateSet = ts
	case RequestPort:

		in, ok := msg.(Request)
		if !ok {
			return fmt.Errorf("invalid input")
		}
		if h.templateSet == nil {
			return fmt.Errorf("template set not loaded")
		}

		buf := &bytes.Buffer{}
		t, ok := h.templateSet[in.Template]
		if !ok {
			err := fmt.Errorf("template not found")
			if !h.settings.EnableErrorPort {
				return err
			}
			return handler(ctx, ErrorPort, Error{
				Context: in.Context,
				Error:   err.Error(),
			})
		}

		err := t.ExecuteTemplate(buf, in.Template, in.RenderData)
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
			Context: in.Context,
			Content: buf.String(),
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
			Label:         "Request",
			Position:      module.Left,
			Source:        true,
			Configuration: Request{},
		},
		{
			Name:          ResponsePort,
			Position:      module.Right,
			Label:         "Response",
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
		settings: defaultEngineSettings,
	}
}

var _ module.Component = (*Component)(nil)

func init() {
	registry.Register(&Component{})
}
