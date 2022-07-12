package main

import (
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
)

const TEMPLATE_DIR string = "./templates/"

type Templates struct {
	templates map[string]*template.Template
}

func (t *Templates) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	temp, ok := t.templates[name]
	if !ok {
		log.Printf("Template %s not found\n", name)
	}
	return temp.Execute(w, data)
}

func isAuth(next echo.HandlerFunc) echo.HandlerFunc {
	return func(ctx echo.Context) error {
		sess, _ := session.Get("store", ctx)
		isAuth, ok := sess.Values["authenticated"]
		if !ok {
			return ctx.Redirect(http.StatusTemporaryRedirect, "/")
		}
		log.Println(isAuth)
		return next(ctx)
	}
}

func forwardAuth(next echo.HandlerFunc) echo.HandlerFunc {
	return func(ctx echo.Context) error {
		sess, _ := session.Get("store", ctx)
		isAuth, ok := sess.Values["authenticated"]
		if ok && isAuth.(bool) {
			return ctx.Redirect(http.StatusTemporaryRedirect, "/restricted")
		}
		return next(ctx)
	}
}

func main() {

	var templates map[string]*template.Template = make(map[string]*template.Template)

	// parse files
	files, err := ioutil.ReadDir(TEMPLATE_DIR)
	if err != nil {
		log.Fatalln(err)
	}

	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".html") {
			templates[file.Name()] = template.Must(template.ParseFiles(TEMPLATE_DIR + file.Name()))
		}
	}

	log.Println(templates)

	app := echo.New()

	f := session.Middleware(sessions.NewCookieStore([]byte("keyboard")))
	app.Use(f)

	app.Renderer = &Templates{
		templates: templates,
	}
	// app.Use(middleware.Logger())
	// app.Use(session.Middleware(sessions.NewCookieStore([]byte("keyboard"))))

	app.GET("/", func(ctx echo.Context) error {
		return ctx.Render(http.StatusOK, "home.html", nil)
	}, forwardAuth)

	app.POST("/login", func(ctx echo.Context) error {
		username := ctx.FormValue("username")
		sess, _ := session.Get("store", ctx)
		sess.Values["authenticated"] = true
		sess.Values["username"] = username
		sess.Options = &sessions.Options{
			Path:     "/",
			MaxAge:   60 * 60,
			Secure:   false,
			HttpOnly: false,
		}
		sess.Save(ctx.Request(), ctx.Response())
		log.Println("Logged in")
		return ctx.Redirect(http.StatusFound, "/restricted")
	})

	app.GET("/restricted", func(ctx echo.Context) error {
		sess, _ := session.Get("store", ctx)
		log.Println(sess.Values)
		username, _ := sess.Values["username"]
		log.Println("Tried to access restricted page")
		return ctx.Render(http.StatusOK, "restricted.html", username)
	}, isAuth)

	app.GET("/logout", func(c echo.Context) error {
		sess, _ := session.Get("store", c)
		sess.Options.MaxAge = -1
		sess.Save(c.Request(), c.Response())
		return c.Redirect(http.StatusTemporaryRedirect, "/")
	})

	log.Fatalln(app.Start(":8000"))
}
