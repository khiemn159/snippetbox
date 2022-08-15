package main

import (
	"errors"
	"strings"
	"fmt"
	"net/http"
	"strconv"
	"sample/snippetbox/pkg/models"
	"sample/snippetbox/pkg/forms"
)

func (app *application) home(w http.ResponseWriter, r *http.Request) {
	s, err := app.snippets.Latest()
	if err != nil {
		app.serverError(w, err)
		return
	}
	app.render(w, r, "home.tmpl", &templateData{Snippets: s})
}

func (app *application) about(w http.ResponseWriter, r *http.Request) {
	app.render(w, r, "about.tmpl", nil)
}

func (app *application) snippetView(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.URL.Query().Get(":id"))
	if err != nil || id < 1 {
		app.notFound(w)
		return
	}

	s, err := app.snippets.Get(id)
	if err != nil {
		if errors.Is(err, models.ErrNoRecord) {
			app.notFound(w)
		} else {
			app.serverError(w, err)
		}
		return

	}

	app.render(w, r, "view.tmpl", &templateData{
		Snippet: s,
	})
}

func (app *application) snippetCreateForm(w http.ResponseWriter, r *http.Request) {
	app.render(w, r, "create.tmpl", &templateData{
		Form: forms.New(nil),
	})
}

func (app *application) snippetCreate(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	form := forms.New(r.PostForm)
	form.Required("title", "content", "expires")
	form.MaxLength("title", 100)
	form.PermittedValues("expires", "1", "7", "365")

	if !form.Valid() {
		app.render(w, r, "create.tmpl", &templateData{Form: form})
		return
	}

	id, err := app.snippets.Insert(form.Get("title"), form.Get("content"), form.Get("expires"))
	if err != nil {
		app.serverError(w, err)
		return
	}

	app.session.Put(r, "flash", "Snippet successfully created!!!")

	http.Redirect(w, r, fmt.Sprintf("/snippet/%d", id), http.StatusSeeOther)
}

func (app *application) signupUserForm(w http.ResponseWriter, r *http.Request) {
	app.render(w, r, "signup.tmpl", &templateData{
		Form: forms.New(nil),
	})
}

func (app *application) signupUser(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	form := forms.New(r.PostForm)
	form.Required("name", "email", "password")
	form.MaxLength("name", 255)
	form.MaxLength("email", 255)
	form.MatchesPattern("email", forms.EmailRX)
	form.MinLength("password", 5)

	if !form.Valid() {
		app.render(w, r, "signup.tmpl", &templateData{Form: form})
		return
	}

	err = app.users.Insert(form.Get("name"), form.Get("email"), form.Get("password"))
	if err != nil {
		if errors.Is(err, models.ErrDuplicateEmail) {
			form.Errors.Add("email", "Address is already in use")
			app.render(w, r, "signup.tmpl", &templateData{Form: form})
		} else {
			app.serverError(w, err)
		}
		return
	}

	app.session.Put(r, "flash", "Your signup was successful! Please login!")

	http.Redirect(w, r, "/user/login", http.StatusSeeOther)
}

func (app *application) loginUserForm(w http.ResponseWriter, r *http.Request) {
	app.render(w, r, "login.tmpl", &templateData{
		Form: forms.New(nil),
	})
}

func (app *application) loginUser(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	form := forms.New(r.PostForm)
	id, err := app.users.Authenticate(form.Get("email"), form.Get("password"))
	if err != nil {
		if errors.Is(err, models.ErrInvalidCredentials) {
			form.Errors.Add("generic", "Email or password incorrect!")
			app.render(w, r, "login.tmpl", &templateData{Form: form})
		} else {
			app.serverError(w, err)
		}
		return
	}

	app.session.Put(r, "authenticatedUserID", id)
	
	relocateurl := app.session.PopString(r, "relocate")
	if relocateurl != "" {
		http.Redirect(w, r, relocateurl, http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/snippet/create", http.StatusSeeOther)
}

func (app *application) logoutUser(w http.ResponseWriter, r *http.Request) {
	app.session.Remove(r, "authenticatedUserID")
	app.session.Put(r, "flash", "You have been logged out successfully!")
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (app *application) profileUser(w http.ResponseWriter, r *http.Request) {
	user, err := app.users.Get(app.session.GetInt(r, "authenticatedUserID"))
	if err != nil {
		app.serverError(w, err)
		return
	}

	app.render(w, r, "profile.tmpl", &templateData{
		User: user,
	})
}

func (app *application) changePasswordForm(w http.ResponseWriter, r *http.Request) {
	app.render(w, r, "change-password.tmpl", &templateData{
		Form: forms.New(nil),
	})
}

func (app *application) changePassword(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	form := forms.New(r.PostForm)
	form.Required("current-password", "new-password", "confirm-password")
	form.MinLength("new-password", 5)

	// check current password
	user, err := app.users.Get(app.session.GetInt(r, "authenticatedUserID"))
	if err != nil {
		app.serverError(w, err)
		return
	}
	_, err = app.users.Authenticate(user.Email, form.Get("current-password"))
	if err != nil {
		if errors.Is(err, models.ErrInvalidCredentials) {
			form.Errors.Add("current-password", "Password incorrect!")
			app.render(w, r, "change-password.tmpl", &templateData{Form: form})
		} else {
			app.serverError(w, err)
		}
		return
	}

	// check new password
	if strings.Compare(form.Get("new-password"), form.Get("confirm-password")) != 0 {
		form.Errors.Add("confirm-password", "Password and confirmation should match!")
	}

	if !form.Valid() {
		app.render(w, r, "change-password.tmpl", &templateData{Form: form})
		return
	}

	// change password
	err = app.users.UpdatePassword(user.ID, form.Get("new-password"))
	if err != nil {
		app.serverError(w, err)
		return
	}

	app.session.Put(r, "flash", "Your password has been changed!")

	http.Redirect(w, r, "/user/profile", http.StatusSeeOther)
}


func ping(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("OK"))
}
	