package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

// UpdateInfo represents the app version information from version.json.
type UpdateInfo struct {
	Version     string `json:"version"`
	BuildNumber int    `json:"build_number"`
	MinVersion  string `json:"min_version"`
	DownloadURL string `json:"download_url"`
	Changelog   string `json:"changelog"`
}

var apkDir string

func init() {
	apkDir = os.Getenv("APK_DIR")
	if apkDir == "" {
		apkDir = "/app/apk"
	}
}

func readVersionFile() (*UpdateInfo, error) {
	path := filepath.Join(apkDir, "version.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var info UpdateInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, err
	}
	return &info, nil
}

// SetupUpdateRoutes registers app update endpoints.
//
//	GET /api/app-info        → version check (min_version for force update)
//	GET /api/update/download → APK download
func SetupUpdateRoutes(r *gin.Engine) {
	api := r.Group("/api")
	{
		api.GET("/app-info", func(c *gin.Context) {
			info, err := readVersionFile()
			if err != nil {
				slog.Warn("version.json not found", "error", err)
				c.JSON(http.StatusNotFound, gin.H{
					"error":   "update_not_available",
					"message": "Version info not found",
				})
				return
			}
			c.JSON(http.StatusOK, info)
		})

		api.GET("/update/download", func(c *gin.Context) {
			apkPath := filepath.Join(apkDir, "vpn-client-android.apk")
			if _, err := os.Stat(apkPath); os.IsNotExist(err) {
				c.JSON(http.StatusNotFound, gin.H{
					"error":   "file_not_found",
					"message": "APK file not found",
				})
				return
			}
			c.Header("Content-Disposition", "attachment; filename=vpn-client-android.apk")
			c.File(apkPath)
		})
	}
}
