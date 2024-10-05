package gui

import (
	"fmt"
	"go/format"
	"io"
	"reflect"
	"strings"

	"github.com/fyne-io/defyne/internal/guidefs"

	"fyne.io/fyne/v2"
)

func ExportGo(obj fyne.CanvasObject, meta map[fyne.CanvasObject]map[string]string, w io.Writer) error {
	guidefs.InitOnce()

	packagesList := packagesRequired(obj, meta)
	varList := varsRequired(obj, meta)
	code := exportCode(packagesList, varList, obj, meta)

	_, err := w.Write([]byte(code))
	return err
}

func ExportGoPreview(obj fyne.CanvasObject, meta map[fyne.CanvasObject]map[string]string, w io.Writer) error {
	guidefs.InitOnce()

	packagesList := packagesRequired(obj, meta)
	packagesList = append(packagesList, "app")
	varList := varsRequired(obj, meta)
	code := exportCode(packagesList, varList, obj, meta)

	code += `
func main() {
	myApp := app.New()
	myWindow := myApp.NewWindow("Hello")
	gui := newGUI()
	myWindow.SetContent(gui.makeUI())
	myWindow.ShowAndRun()
}
`
	_, err := w.Write([]byte(code))

	return err
}

func exportCode(pkgs, vars []string, obj fyne.CanvasObject, meta map[fyne.CanvasObject]map[string]string) string {
	for i := 0; i < len(pkgs); i++ {
		if pkgs[i] != "net/url" {
			pkgs[i] = "fyne.io/fyne/v2/" + pkgs[i]
		}

		pkgs[i] = fmt.Sprintf(`	"%s"`, pkgs[i])
	}

	defs := make(map[string]string)

	_, clazz := getTypeOf(obj)
	main := guidefs.Lookup(clazz).Gostring(obj, meta, defs)
	setup := ""
	for k, v := range defs {
		setup += "g." + k + " = " + v + "\n"
	}

	code := fmt.Sprintf(`// auto-generated
// Code generated by GUI builder.

package main

import (
	"fyne.io/fyne/v2"
%s
)

type gui struct {
%s
}

func newGUI() *gui {
	return &gui{}
}

func (g *gui) makeUI() fyne.CanvasObject {
	%s

	return %s}
`,
		strings.Join(pkgs, "\n"),
		strings.Join(vars, "\n"),
		setup, main)

	formatted, err := format.Source([]byte(code))
	if err != nil {
		fyne.LogError("Failed to format GUI code", err)
		return code
	}
	return string(formatted)
}

func packagesRequired(obj fyne.CanvasObject, meta map[fyne.CanvasObject]map[string]string) []string {
	ret := []string{"container"}
	var objs []fyne.CanvasObject
	if c, ok := obj.(*fyne.Container); ok {
		objs = c.Objects
		layout, ok := meta[c]["layout"]
		if ok && layout == "Form" {
			ret = append(ret, "layout")
		}
	} else {
		class := reflect.TypeOf(obj).String()
		info := guidefs.Lookup(class)

		if info != nil && info.IsContainer() {
			objs = info.Children(obj)
		} else {
			if w, ok := obj.(fyne.Widget); ok {
				return packagesRequiredForWidget(w)
			}
		}
	}

	for _, w := range objs {
		for _, p := range packagesRequired(w, meta) {
			added := false
			for _, exists := range ret {
				if p == exists {
					added = true
					break
				}
			}
			if !added {
				ret = append(ret, p)
			}
		}
	}
	return ret
}

func packagesRequiredForWidget(w fyne.Widget) []string {
	name := reflect.TypeOf(w).String()
	if pkgs := guidefs.Lookup(name).Packages; pkgs != nil {
		return pkgs(w)
	}

	return []string{"widget"}
}

func varsRequired(obj fyne.CanvasObject, props map[fyne.CanvasObject]map[string]string) []string {
	name := props[obj]["name"]

	var ret []string
	if c, ok := obj.(*fyne.Container); ok {
		if name != "" {
			ret = append(ret, name+" "+"*fyne.Container")
		}

		for _, w := range c.Objects {
			ret = append(ret, varsRequired(w, props)...)
		}
	} else {
		class := reflect.TypeOf(obj).String()
		info := guidefs.Lookup(class)

		if info != nil && info.IsContainer() {
			for _, child := range info.Children(obj) {
				ret = append(ret, varsRequired(child, props)...)
			}
		}
	}

	if w, ok := obj.(fyne.Widget); ok {
		if name != "" {
			_, class := getTypeOf(w)
			ret = append(ret, name+" "+class)
		}
	}
	return ret
}
