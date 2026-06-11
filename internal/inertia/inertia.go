package inertia

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/petaki/inertia-go"
	"github.com/petaki/support-go/vite"
	"github.com/tiar/telegram-sender/internal/config"
)

func PublicHandler(publicFS embed.FS) http.Handler {
	sub, err := fs.Sub(publicFS, "public")
	if err != nil {
		log.Fatalf("cannot sub public fs: %s", err)
	}
	return http.FileServer(http.FS(sub))
}
func New(cfg config.Config, publicFS embed.FS, viewFS embed.FS) *inertia.Inertia {
	var viteManager *vite.Vite
	subFS, err := fs.Sub(publicFS, "public") // strip public
	if _, err := os.Stat("public/hot"); err == nil {
		viteManager = vite.New("public", "build")
		fmt.Println("Running in development mode")
	} else {
		viteManager = vite.New("public", "build", subFS)
		fmt.Println("Running in production mode")
	}

	version, err := viteManager.ManifestHash()
	if err != nil {
		version = "dev"
	}

	i := inertia.New(cfg.AppURL, "resources/views/root.html", version, viewFS)
	i.ShareFunc("isRunningHot", viteManager.IsRunningHot)
	i.ShareFunc("asset", viteManager.Asset)
	i.ShareFunc("css", viteManager.CSS)
	i.ShareFunc("viteDevUrl", func(entry string) string {
		content, err := os.ReadFile("public/hot")
		if err != nil {
			return "//localhost:5173/" + strings.TrimPrefix(entry, "/")
		}
		base := strings.TrimSpace(string(content))
		return base + "/" + strings.TrimPrefix(entry, "/")
	})

	return i
}
