package main

import (
	"controlp/internal/auth"
	"controlp/internal/blueprints"
	"controlp/internal/cron"
	"controlp/internal/db"
	"controlp/internal/firewall"
	internalfs "controlp/internal/fs"
	"controlp/internal/logs"
	"controlp/internal/nginx"
	"controlp/internal/security"
	"controlp/internal/ssl"
	"controlp/internal/system"
	"controlp/internal/terminal" // NEW: Terminal Logic
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/contrib/websocket" // NEW: WebSocket Support
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	"github.com/gofiber/template/html/v2"
)

//go:embed views/* public/*
var embedDir embed.FS

func main() {
	// ---------------------------------------------------------
	// 1. Initialize Systems
	// ---------------------------------------------------------
	auth.Init()

	// ---------------------------------------------------------
	// 2. Prepare View Engine
	// ---------------------------------------------------------
	viewsFS, err := fs.Sub(embedDir, "views")
	if err != nil {
		log.Fatal(err)
	}
	engine := html.NewFileSystem(http.FS(viewsFS), ".html")

	// ---------------------------------------------------------
	// 3. Setup Fiber App
	// ---------------------------------------------------------
	app := fiber.New(fiber.Config{
		Views:       engine,
		ViewsLayout: "layouts/main",
	})

	// ---------------------------------------------------------
	// Global Middleware
	// ---------------------------------------------------------

	// NEW: WebSocket Upgrade Middleware
	app.Use("/ws", func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			c.Locals("allowed", true)
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})

	app.Use(func(c *fiber.Ctx) error {
		c.Set("Cache-Control", "no-cache, no-store, must-revalidate")
		c.Set("Pragma", "no-cache")
		c.Set("Expires", "0")
		return c.Next()
	})

	app.Use(func(c *fiber.Ctx) error {
		if strings.HasPrefix(c.Path(), "/api/") {
			fmt.Printf("[HTTP] %s %s\n", c.Method(), c.Path())
		}
		return c.Next()
	})

	// ---------------------------------------------------------
	// 4. Public Routes
	// ---------------------------------------------------------
	publicFS, err := fs.Sub(embedDir, "public")
	if err != nil {
		log.Fatal(err)
	}
	app.Use("/public", filesystem.New(filesystem.Config{
		Root:   http.FS(publicFS),
		Browse: false,
	}))

	app.Get("/login", func(c *fiber.Ctx) error {
		return c.Render("login", fiber.Map{}, "layouts/empty")
	})

	app.Post("/login", func(c *fiber.Ctx) error {
		user := c.FormValue("username")
		pass := c.FormValue("password")
		if auth.CheckCredentials(user, pass) {
			sess, _ := auth.Store.Get(c)
			sess.Set("authenticated", true)
			sess.Save()
			return c.Redirect("/")
		}
		return c.Render("login", fiber.Map{"Error": "Invalid Credentials"}, "layouts/empty")
	})

	app.Get("/logout", func(c *fiber.Ctx) error {
		sess, _ := auth.Store.Get(c)
		sess.Destroy()
		return c.Redirect("/login")
	})

	// ---------------------------------------------------------
	// 5. Protected Routes
	// ---------------------------------------------------------
	app.Use(auth.Protected())

	// --- DASHBOARD ---
	app.Get("/", func(c *fiber.Ctx) error {
		stats := system.GetStats()
		return c.Render("dashboard", fiber.Map{
			"Title": "Overview",
			"Stats": stats,
		})
	})

	// --- WEBSITES (NGINX) ---
	app.Get("/sites", func(c *fiber.Ctx) error {
		sites, err := nginx.ListSites()
		if err != nil {
			return c.Status(500).SendString("Error listing sites")
		}
		return c.Render("sites", fiber.Map{
			"Title": "Domains",
			"Sites": sites,
		})
	})

	app.Get("/api/sites/list", func(c *fiber.Ctx) error {
		sites, err := nginx.ListSites()
		if err != nil {
			return c.Status(500).SendString("Error listing sites")
		}
		return c.Render("partials/site_rows", fiber.Map{"Sites": sites}, "layouts/empty")
	})

	app.Get("/api/sites/options", func(c *fiber.Ctx) error {
		sites, _ := nginx.ListSites()
		html := ""
		for _, s := range sites {
			html += fmt.Sprintf(`<option value="%s">%s</option>`, s.Domain, s.Domain)
		}
		return c.SendString(html)
	})

	app.Post("/api/sites/create", func(c *fiber.Ctx) error {
		err := nginx.CreateSite(c.FormValue("domain"))
		if err != nil {
			return c.SendString(err.Error())
		}
		sites, _ := nginx.ListSites()
		return c.Render("partials/site_rows", fiber.Map{"Sites": sites}, "layouts/empty")
	})

	app.Post("/api/sites/delete", func(c *fiber.Ctx) error {
		err := nginx.DeleteSite(c.Query("domain"))
		if err != nil {
			return c.SendString(err.Error())
		}
		sites, _ := nginx.ListSites()
		return c.Render("partials/site_rows", fiber.Map{"Sites": sites}, "layouts/empty")
	})

	app.Post("/api/sites/ssl", func(c *fiber.Ctx) error {
		domain := c.Query("domain")
		if err := ssl.EnableSSL(domain, "admin@"+domain); err != nil {
			return c.Status(500).SendString("SSL Error: " + err.Error())
		}
		sites, _ := nginx.ListSites()
		return c.Render("partials/site_rows", fiber.Map{"Sites": sites}, "layouts/empty")
	})

	// --- BLUEPRINTS (APP STORE) ---
	app.Get("/blueprints", func(c *fiber.Ctx) error {
		return c.Render("blueprints", fiber.Map{
			"Title": "App Blueprints",
			"Apps":  blueprints.List(),
		})
	})

	app.Post("/api/blueprints/install", func(c *fiber.Ctx) error {
		appID := c.FormValue("app_id")
		domain := strings.TrimSpace(c.FormValue("domain"))
		dbSuffix := strings.TrimSpace(c.FormValue("db_suffix"))
		userSuffix := strings.TrimSpace(c.FormValue("user_suffix"))
		rootPass := c.FormValue("root_password")

		if domain == "" {
			return c.SendString(`<div class="bg-red-50 text-vn-danger p-4 rounded-xl border border-red-100 font-medium">Please select or enter a domain.</div>`)
		}

		rootPath := filepath.Join(blueprints.WebRoot, domain)
		if _, err := os.Stat(rootPath); os.IsNotExist(err) {
			fmt.Println("[Main] Domain folder missing, creating Nginx site for:", domain)
			if err := nginx.CreateSite(domain); err != nil {
				return c.SendString(fmt.Sprintf(`<div class="bg-red-50 text-vn-danger p-4 rounded-xl border border-red-100 font-medium">Failed to create domain structure: %s</div>`, err.Error()))
			}
		}

		time.Sleep(500 * time.Millisecond)

		progressLogger := func(step string, percent int) {
			fmt.Printf("[Blueprint: %s] %d%% - %s\n", domain, percent, step)
		}

		var err error
		switch appID {
		case "wordpress":
			err = blueprints.InstallWordPress(domain, dbSuffix, userSuffix, rootPass, progressLogger)
		case "laravel":
			err = blueprints.InstallLaravel(domain, rootPass, progressLogger)
		default:
			return c.SendString(`<div class="text-vn-danger">Unknown App ID</div>`)
		}

		if err != nil {
			return c.SendString(fmt.Sprintf(`
                <div class="bg-red-50 border border-red-100 p-6 rounded-xl shadow-sm">
                    <div class="flex items-start gap-4">
                        <div class="w-10 h-10 rounded-full bg-red-100 flex items-center justify-center shrink-0">
                            <i class="ph-bold ph-warning text-vn-danger text-xl"></i>
                        </div>
                        <div>
                            <h4 class="text-vn-danger font-bold text-lg mb-1">Installation Failed</h4>
                            <p class="text-gray-600 text-sm mb-4">%s</p>
                            <button @click="modalOpen = false" class="text-xs font-bold text-gray-500 hover:text-vn-dark underline">Close</button>
                        </div>
                    </div>
                </div>
            `, err.Error()))
		}

		return c.SendString(fmt.Sprintf(`
            <div class="bg-green-50 border border-green-100 p-8 rounded-xl shadow-sm text-center animate-fade-in">
                <div class="w-16 h-16 rounded-full bg-green-100 flex items-center justify-center mx-auto mb-4">
                    <i class="ph-fill ph-check-circle text-vn-success text-3xl"></i>
                </div>
                <h3 class="text-2xl font-bold text-vn-success mb-2">Success!</h3>
                <p class="text-gray-600 mb-6">
                    <span class="font-semibold text-vn-dark">%s</span> has been successfully deployed to <span class="font-semibold text-vn-dark">%s</span>.
                </p>
                
                <div class="bg-white border border-green-100 rounded-lg p-4 text-left max-w-sm mx-auto mb-6 shadow-sm">
                    <div class="flex justify-between items-center mb-2 pb-2 border-b border-gray-50">
                        <span class="text-xs text-gray-400 font-bold uppercase">Database</span>
                        <span class="text-sm font-mono text-gray-600">db_%s</span>
                    </div>
                    <div class="flex justify-between items-center">
                        <span class="text-xs text-gray-400 font-bold uppercase">User</span>
                        <span class="text-sm font-mono text-gray-600">user_%s</span>
                    </div>
                </div>

                <div class="flex justify-center gap-3">
                    <a href="http://%s" target="_blank" class="px-6 py-2.5 rounded-xl bg-vn-success text-white font-medium hover:bg-green-600 transition shadow-lg shadow-green-100">
                        Visit Website <i class="ph-bold ph-arrow-right ml-1"></i>
                    </a>
                </div>
            </div>
        `, appID, domain, dbSuffix, userSuffix, domain))
	})

	// --- FILE MANAGER ---
	app.Get("/files", func(c *fiber.Ctx) error {
		return c.Render("files", fiber.Map{"Title": "File Manager"})
	})

	app.Get("/api/files/tree", func(c *fiber.Ctx) error {
		path := c.Query("path", "")
		entries, err := internalfs.ListDir(path)
		if err != nil {
			return c.Status(500).SendString("Error loading folder")
		}
		return c.Render("partials/file_tree", fiber.Map{"Entries": entries}, "layouts/empty")
	})

	app.Get("/api/files/read", func(c *fiber.Ctx) error {
		content, err := internalfs.ReadFile(c.Query("path"))
		if err != nil {
			return c.Status(500).SendString("Error reading file")
		}
		return c.SendString(content)
	})

	app.Post("/api/files/save", func(c *fiber.Ctx) error {
		err := internalfs.SaveFile(c.FormValue("path"), c.FormValue("content"))
		if err != nil {
			return c.Status(500).SendString("Error saving file")
		}
		return c.SendString("Saved!")
	})

	app.Post("/api/files/create", func(c *fiber.Ctx) error {
		path := c.FormValue("path")
		name := c.FormValue("name")
		isFolder := c.FormValue("is_folder") == "true"
		err := internalfs.CreateItem(path, name, isFolder)
		if err != nil {
			return c.Status(500).SendString("Error")
		}
		return c.SendString("Created")
	})

	app.Post("/api/files/delete", func(c *fiber.Ctx) error {
		path := c.FormValue("path")
		err := internalfs.DeleteItem(path)
		if err != nil {
			return c.Status(500).SendString("Error")
		}
		return c.SendString("Deleted")
	})

	app.Post("/api/files/upload", func(c *fiber.Ctx) error {
		path := c.FormValue("path")
		file, err := c.FormFile("file")
		if err != nil {
			return c.Status(400).SendString("No file")
		}
		err = internalfs.UploadFile(path, file)
		if err != nil {
			return c.Status(500).SendString("Error")
		}
		return c.SendString("Uploaded")
	})

	// --- CRON JOBS ---
	app.Get("/cron", func(c *fiber.Ctx) error {
		jobs, _ := cron.ListJobs()
		return c.Render("cron", fiber.Map{"Title": "Task Scheduler", "Jobs": jobs})
	})

	app.Get("/api/cron/list", func(c *fiber.Ctx) error {
		jobs, _ := cron.ListJobs()
		return c.Render("partials/cron_rows", fiber.Map{"Jobs": jobs}, "layouts/empty")
	})

	app.Post("/api/cron/create", func(c *fiber.Ctx) error {
		cron.AddJob(c.FormValue("schedule"), c.FormValue("command"))
		jobs, _ := cron.ListJobs()
		return c.Render("partials/cron_rows", fiber.Map{"Jobs": jobs}, "layouts/empty")
	})

	app.Post("/api/cron/delete", func(c *fiber.Ctx) error {
		id, _ := strconv.Atoi(c.Query("id"))
		cron.DeleteJob(id)
		jobs, _ := cron.ListJobs()
		return c.Render("partials/cron_rows", fiber.Map{"Jobs": jobs}, "layouts/empty")
	})

	// --- DATABASE ---
	app.Get("/database", func(c *fiber.Ctx) error {
		return c.Render("database", fiber.Map{"Title": "Database"})
	})

	app.Get("/api/database/pma-status", func(c *fiber.Ctx) error {
		installed := db.CheckPhpMyAdmin()
		if installed {
			return c.SendString(`<a href="#" onclick="window.open(window.location.protocol + '//' + window.location.hostname + '/phpmyadmin', '_blank'); return false;" class="bg-vn-gray text-vn-dark hover:bg-vn-primary hover:text-white px-4 py-2 rounded-pill text-sm font-medium flex items-center gap-2 transition shadow-sm"><i class="ph ph-database"></i> Open phpMyAdmin</a>`)
		}
		return c.SendString(`<button hx-post="/api/database/pma-install" hx-swap="outerHTML" class="bg-vn-primary text-white hover:bg-purple-600 px-4 py-2 rounded-pill text-sm font-medium flex items-center gap-2 transition shadow-md shadow-purple-200"><i class="ph ph-download-simple"></i> Install phpMyAdmin</button>`)
	})

	app.Post("/api/database/pma-install", func(c *fiber.Ctx) error {
		err := db.InstallPhpMyAdmin()
		if err != nil {
			return c.SendString(`<div class="text-vn-danger text-sm">Installation Failed: ` + err.Error() + `</div>`)
		}

		pmaPath := filepath.Join(blueprints.WebRoot, "phpmyadmin")
		matches, _ := filepath.Glob(filepath.Join(pmaPath, "phpMyAdmin-*-all-languages"))
		if len(matches) > 0 {
			subDir := matches[0]
			exec.Command("sh", "-c", fmt.Sprintf("mv %s/* %s/", subDir, pmaPath)).Run()
			exec.Command("sh", "-c", fmt.Sprintf("mv %s/.* %s/", subDir, pmaPath)).Run()
			os.RemoveAll(subDir)
		}

		if runtime.GOOS == "linux" {
			exec.Command("chown", "-R", "www-data:www-data", pmaPath).Run()
			exec.Command("chmod", "-R", "755", pmaPath).Run()
		}

		return c.SendString(`<a href="#" onclick="window.open(window.location.protocol + '//' + window.location.hostname + '/phpmyadmin', '_blank'); return false;" class="bg-vn-gray text-vn-dark hover:bg-vn-primary hover:text-white px-4 py-2 rounded-pill text-sm font-medium flex items-center gap-2 transition shadow-sm"><i class="ph ph-database"></i> Open phpMyAdmin</a>`)
	})

	app.Post("/api/database/create", func(c *fiber.Ctx) error {
		name := "db_" + c.FormValue("db_name")
		user := "user_" + c.FormValue("db_user")
		result, err := db.CreateDatabase(name, user)
		if err != nil {
			return c.SendString(err.Error())
		}
		return c.SendString(fmt.Sprintf(`
            <div class="bg-green-50 border border-green-100 p-4 rounded-xl animate-fade-in">
                <div class="text-vn-success font-bold text-sm mb-1 uppercase tracking-wider">Database Provisioned</div>
                <div class="space-y-1 font-mono text-xs text-gray-600">
                    <p>DB: %s</p>
                    <p>User: %s</p>
                    <p>Pass: %s</p>
                </div>
            </div>
        `, result.DBName, result.Username, result.Password))
	})

	// --- LOGS ---
	app.Get("/logs", func(c *fiber.Ctx) error {
		return c.Render("logs", fiber.Map{"Title": "Logs"})
	})

	app.Get("/api/logs/read", func(c *fiber.Ctx) error {
		content, err := logs.ReadLog(c.Query("type"))
		if err != nil {
			return c.SendString("Error: " + err.Error())
		}
		return c.SendString(content)
	})

	// --- SETTINGS ---
	app.Get("/settings", func(c *fiber.Ctx) error {
		return c.Render("settings", fiber.Map{"Title": "Server Settings"})
	})

	app.Post("/api/service/:name/restart", func(c *fiber.Ctx) error {
		err := system.RestartService(c.Params("name"))
		if err != nil {
			return c.SendString("Error")
		}
		return c.SendString("Done")
	})

	// --- FIREWALL ---
	app.Get("/firewall", func(c *fiber.Ctx) error {
		rules, _ := firewall.ListRules()
		return c.Render("firewall", fiber.Map{"Title": "Firewall", "Rules": rules})
	})

	app.Post("/api/firewall/add", func(c *fiber.Ctx) error {
		firewall.AddRule(c.FormValue("port"), c.FormValue("proto"))
		rules, _ := firewall.ListRules()
		return c.Render("partials/firewall_rows", fiber.Map{"Rules": rules}, "layouts/empty")
	})

	app.Post("/api/firewall/delete", func(c *fiber.Ctx) error {
		firewall.DeleteRule(c.Query("port"), c.Query("proto"))
		rules, _ := firewall.ListRules()
		return c.Render("partials/firewall_rows", fiber.Map{"Rules": rules}, "layouts/empty")
	})

	// --- ENVIRONMENT MANAGER ---
	app.Get("/env", func(c *fiber.Ctx) error {
		content, err := embedDir.ReadFile("views/env.html")
		if err != nil {
			return c.Status(500).SendString("Error loading environment manager: " + err.Error())
		}
		c.Set("Content-Type", "text/html")
		return c.Send(content)
	})

	// --- SYSTEM UPDATES ---
	app.Get("/updates", func(c *fiber.Ctx) error {
		return c.Render("updates", fiber.Map{"Title": "System Updates"})
	})

	app.Get("/api/updates/list", func(c *fiber.Ctx) error {
		pkgs, _ := system.CheckUpdates()
		return c.Render("partials/update_rows", fiber.Map{"Packages": pkgs}, "layouts/empty")
	})

	app.Post("/api/updates/upgrade", func(c *fiber.Ctx) error {
		if err := system.RunUpgrade(); err != nil {
			return c.SendString(fmt.Sprintf(`<div class="bg-red-50 border border-red-100 text-vn-danger p-4 rounded-xl text-sm font-medium flex items-center gap-2"><i class="ph-bold ph-warning"></i> Upgrade Failed: %s</div>`, err.Error()))
		}
		return c.SendString(`<div class="bg-green-50 border border-green-100 text-vn-success p-4 rounded-xl text-sm font-medium flex items-center gap-2"><i class="ph-bold ph-check-circle"></i> System Upgraded Successfully!</div>`)
	})

	// --- FAIL2BAN ---
	app.Get("/fail2ban", func(c *fiber.Ctx) error {
		return c.Render("fail2ban", fiber.Map{"Title": "Intrusion Prevention"})
	})

	app.Get("/api/security/fail2ban/list", func(c *fiber.Ctx) error {
		jails, err := security.ListJails()
		if err != nil {
			return c.Status(500).SendString(`<div class="text-red-500 p-6">Error: ` + err.Error() + `</div>`)
		}
		return c.Render("partials/fail2ban_rows", fiber.Map{"Jails": jails}, "layouts/empty")
	})

	app.Post("/api/security/fail2ban/unban", func(c *fiber.Ctx) error {
		security.UnbanIP(c.Query("jail"), c.Query("ip"))
		jails, _ := security.ListJails()
		return c.Render("partials/fail2ban_rows", fiber.Map{"Jails": jails}, "layouts/empty")
	})

	// --- TERMINAL ---
	// Serve the terminal view (Standalone Page)
	app.Get("/terminal", func(c *fiber.Ctx) error {
		return c.Render("terminal", fiber.Map{}, "layouts/empty")
	})

	// WebSocket handler for the terminal
	app.Get("/ws/terminal", websocket.New(terminal.Handler))

	log.Println("ControlP starting on :8888")
	log.Fatal(app.Listen(":8888"))
}
