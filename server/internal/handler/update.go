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

// AppInfo returns the latest app version information for in-app update checks.
// @Summary      App version info
// @Description  Returns the latest app version, minimum required version, and changelog for in-app update checks
// @Tags         Public
// @Success      200 {object} handler.UpdateInfo "Version info"
// @Failure      404 {object} map[string]string "Version info not found"
// @Router       /api/app-info [get]
func AppInfo(c *gin.Context) {
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
}

// DownloadApk serves the Android APK file for download.
// @Summary      Download APK
// @Description  Downloads the latest Android APK for in-app update
// @Tags         Public
// @Success      200 {file} binary "APK file"
// @Failure      404 {object} map[string]string "APK not found"
// @Router       /api/update/download [get]
func DownloadApk(c *gin.Context) {
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
}

func SetupUpdateRoutes(r *gin.Engine) {
	api := r.Group("/api")
	{
		api.GET("/app-info", AppInfo)
		api.GET("/update/download", DownloadApk)
	}
}
