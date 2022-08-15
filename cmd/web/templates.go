package main

import (
	"html/template"
	"path/filepath"
	"sample/snippetbox/pkg/models"
	"sample/snippetbox/pkg/forms"
	"time"
)

type templateData struct {
	CSRFToken       string
	CurrentYear     int
	Flash           string
	Form            *forms.Form
	IsAuthenticated bool
	User            *models.User
	Snippet         *models.Snippet
	Snippets        []*models.Snippet
}

func humanDate(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format("02 Jan 2006 at 15:04")
}

var functions = template.FuncMap{
	"humanDate": humanDate,
}

func newTemplateCache(dir string) (map[string]*template.Template, error) {
	cache := map[string]*template.Template{}
	
	pages, err := filepath.Glob(filepath.Join(dir, "pages/*.tmpl"))
	if err != nil {
		return nil, err
	}
	
	for _, page := range pages {
		name := filepath.Base(page)
		ts, err := template.New(name).Funcs(functions).ParseFiles(page)
		if err != nil {
			return nil, err
		}
		
		ts, err = ts.ParseGlob(filepath.Join(dir, "*.tmpl"))
		if err != nil {
			return nil, err
		}
		
		ts, err = ts.ParseGlob(filepath.Join(dir, "partials/*.tmpl"))
		if err != nil {
			return nil, err
		}
		// Add the template set to the cache, using the name of the page
		// (like 'home.tmpl') as the key.
		cache[name] = ts
	}
	// Return the map.
	return cache, nil
}
