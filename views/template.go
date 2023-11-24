package views

import (
	"bytes"
	"embed"
	"html/template"
	"io"

	"github.com/davidbanham/scum/components"
)

//go:embed pages/*.html layouts/*.html components/*.html
var FS embed.FS

// Tmpl exports the compiled templates
type Templater struct {
	tmpl       map[string]*template.Template
	components *template.Template
	root       *template.Template
}

func Tmpl(funcMap template.FuncMap) (Templater, error) {
	t := Templater{}
	t.tmpl = map[string]*template.Template{}
	base, err := components.Tmpl()
	if err != nil {
		return t, err
	}
	components, err := base.Funcs(funcMap).ParseFS(FS, "components/*", "layouts/*")
	if err != nil {
		return t, err
	}
	root, err := components.Clone()
	if err != nil {
		return t, err
	}

	t.root = root
	t.components = components

	return t, nil
}

func (this *Templater) ExecuteTemplate(w io.Writer, filename string, data interface{}) error {
	if _, ok := this.tmpl[filename]; !ok {
		clone, err := this.root.Clone()
		if err != nil {
			return err
		}
		parsed, err := clone.ParseFS(FS, "pages/"+filename)
		if err != nil {
			return nil
		}
		this.tmpl[filename] = parsed
	}

	return this.tmpl[filename].ExecuteTemplate(w, filename, data)
}

func (this *Templater) FillCache() error {
	files, err := FS.ReadDir("pages")
	if err != nil {
		return err
	}
	for _, f := range files {
		filename := f.Name()
		if _, ok := this.tmpl[filename]; !ok {
			clone, err := this.root.Clone()
			if err != nil {
				return err
			}
			parsed, err := clone.ParseFS(FS, "pages/"+filename)
			if err != nil {
				return nil
			}
			this.tmpl[filename] = parsed
		}
	}
	return nil
}

func (this Templater) Component(name string, data interface{}) (template.HTML, error) {
	buf := bytes.NewBuffer([]byte{})
	if err := this.components.ExecuteTemplate(buf, name, data); err != nil {
		return template.HTML(""), err
	}
	return template.HTML(buf.String()), nil
}
