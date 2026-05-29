package main

import (
	"fmt"
	"html/template"
	"io"
	"log"
	"os"
	"path/filepath"

	"golang-dashboard/internal/database"
	"golang-dashboard/internal/models"
	"golang-dashboard/internal/routes"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
)

type TemplateRenderer struct {
	templates map[string]*template.Template
}

func (t *TemplateRenderer) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	tmpl, ok := t.templates[name]
	if !ok {
		return fmt.Errorf("template %q is not registered", name)
	}

	return tmpl.ExecuteTemplate(w, name, data)
}

func main() {
	_ = godotenv.Load()

	database.Connect()

	if database.DB != nil {
		database.DB.AutoMigrate(
			&models.Cluster{},
			&models.User{},
			&models.Request{},
		)
		ensureWorkflowConstraints()
	}

	e := echo.New()

	renderer := &TemplateRenderer{
		templates: mustParseTemplates(),
	}

	e.Renderer = renderer
	e.Static("/static", "web/static")
	e.Static("/truck_label", "web/truck_label")

	routes.RegisterRoutes(e)

	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8080"
	}
	host := os.Getenv("APP_HOST")
	if host == "" {
		host = "127.0.0.1"
	}

	addr := host + ":" + port
	log.Println("Server running on", addr)
	e.Logger.Fatal(e.Start(addr))
}

func ensureWorkflowConstraints() {
	statuses := "'PENDING_OPS', 'PENDING_MM', 'ASSIGNED', 'FOR_DOCKING', 'DOCKED', 'CONFIRMED', 'REJECTED', 'CANCELED'"
	if err := database.DB.Exec(`ALTER TABLE requests DROP CONSTRAINT IF EXISTS requests_status_check`).Error; err != nil {
		log.Println("Unable to drop request status constraint:", err)
	}
	if err := database.DB.Exec(fmt.Sprintf(`ALTER TABLE requests ADD CONSTRAINT requests_status_check CHECK (status IN (%s))`, statuses)).Error; err != nil {
		log.Println("Unable to add request status constraint:", err)
	}

	roles := "'fte_ops', 'fte_mm', 'ops_pic', 'dock_officer', 'doc_officer', 'data_team', 'admin'"
	if err := database.DB.Exec(`
DO $$
DECLARE
	role_constraint text;
BEGIN
	FOR role_constraint IN
		SELECT conname
		FROM pg_constraint
		WHERE conrelid = 'users'::regclass
			AND contype = 'c'
			AND pg_get_constraintdef(oid) ILIKE '%role%'
	LOOP
		EXECUTE format('ALTER TABLE users DROP CONSTRAINT %I', role_constraint);
	END LOOP;
END $$;
`).Error; err != nil {
		log.Println("Unable to drop user role constraint:", err)
	}
	if err := database.DB.Exec(fmt.Sprintf(`ALTER TABLE users ADD CONSTRAINT users_role_check CHECK (role IN (%s))`, roles)).Error; err != nil {
		log.Println("Unable to add user role constraint:", err)
	}
}

func mustParseTemplates() map[string]*template.Template {
	pages, err := filepath.Glob("web/templates/*.html")
	if err != nil {
		log.Fatal("Template discovery failed:", err)
	}

	templates := make(map[string]*template.Template)
	layout := filepath.Join("web", "templates", "layout.html")

	for _, page := range pages {
		name := filepath.Base(page)
		if name == "layout.html" {
			continue
		}

		templates[name] = template.Must(template.New("layout.html").Funcs(template.FuncMap{
			"assetVersion": assetVersion,
			"add": func(a, b int64) int64 {
				return a + b
			},
		}).ParseFiles(layout, page))
	}

	return templates
}

func assetVersion(path string) string {
	info, err := os.Stat(path)
	if err != nil {
		return "dev"
	}

	return fmt.Sprintf("%d", info.ModTime().Unix())
}
