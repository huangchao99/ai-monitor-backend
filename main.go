package main

import (
	"log"
	"net/http"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"ai-monitor-backend/api"
	"ai-monitor-backend/config"
	"ai-monitor-backend/pyservice"
	"ai-monitor-backend/store"
	"ai-monitor-backend/zlm"
)

func main() {
	// Open DB
	s, err := store.New(config.DBPath)
	if err != nil {
		log.Fatalf("DB open failed: %v", err)
	}
	log.Printf("DB opened: %s", config.DBPath)

	// Handlers
	cameraH := api.NewCameraHandler(s)
	taskH := api.NewTaskHandler(s)
	alarmH := api.NewAlarmHandler(s)
	algoMgmtH := api.NewAlgoManageHandler(s)
	voiceAlarmH := api.NewVoiceAlarmHandler(s)

	// Gin router
	r := gin.Default()

	// CORS — allow all origins for development
	r.Use(cors.New(cors.Config{
		AllowAllOrigins:  true,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: false,
	}))

	// Health
	r.GET("/api/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"code":    0,
			"message": "ok",
			"data": gin.H{
				"zlm_alive":    zlm.ServerStatus(),
				"python_alive": pyservice.IsHealthy(),
			},
		})
	})

	// Cameras
	cams := r.Group("/api/cameras")
	{
		cams.GET("", cameraH.List)
		cams.POST("", cameraH.Create)
		cams.PUT("/:id", cameraH.Update)
		cams.DELETE("/:id", cameraH.Delete)
		cams.POST("/:id/stream/start", cameraH.StreamStart)
		cams.POST("/:id/stream/stop", cameraH.StreamStop)
		cams.GET("/:id/snapshot", cameraH.Snapshot)
	}

	// Algorithms (read-only for task page, uses existing handler)
	r.GET("/api/algorithms", taskH.ListAlgorithms)

	// Algorithm management (CRUD + model association)
	algoMgmt := r.Group("/api/algo-manage")
	{
		algoMgmt.GET("/algorithms", algoMgmtH.ListAlgorithms)
		algoMgmt.POST("/algorithms", algoMgmtH.CreateAlgorithm)
		algoMgmt.PUT("/algorithms/:id", algoMgmtH.UpdateAlgorithm)
		algoMgmt.DELETE("/algorithms/:id", algoMgmtH.DeleteAlgorithm)

		algoMgmt.GET("/models", algoMgmtH.ListModels)
		algoMgmt.POST("/models", algoMgmtH.CreateModel)
		algoMgmt.PUT("/models/:id", algoMgmtH.UpdateModel)
		algoMgmt.DELETE("/models/:id", algoMgmtH.DeleteModel)

		algoMgmt.GET("/plugins", algoMgmtH.ListPlugins)
		algoMgmt.POST("/plugins", algoMgmtH.UploadPlugin)
		algoMgmt.DELETE("/plugins/:filename", algoMgmtH.DeletePlugin)

		algoMgmt.POST("/upload-file", algoMgmtH.UploadModelFile)
	}

	// Tasks
	tasks := r.Group("/api/tasks")
	{
		tasks.GET("", taskH.List)
		tasks.POST("", taskH.Create)
		tasks.PUT("/:id", taskH.Update)
		tasks.DELETE("/:id", taskH.Delete)
		tasks.POST("/:id/start", taskH.Start)
		tasks.POST("/:id/stop", taskH.Stop)
	}

	// Alarms
	alarms := r.Group("/api/alarms")
	{
		alarms.GET("", alarmH.List)
		alarms.PUT("/:id", alarmH.UpdateStatus)
		alarms.DELETE("/:id", alarmH.Delete)
		alarms.POST("/batch-delete", alarmH.BatchDelete)
	}

	// Voice alarm management
	va := r.Group("/api/voice-alarm")
	{
		va.GET("/settings", voiceAlarmH.GetSettings)
		va.PUT("/settings", voiceAlarmH.SaveSettings)
		va.GET("/algo-map", voiceAlarmH.ListAlgoMap)
		va.PUT("/algo-map/:algo_id", voiceAlarmH.SetAlgoMap)
		va.DELETE("/algo-map/:algo_id", voiceAlarmH.DeleteAlgoMap)
		va.GET("/audio-files", voiceAlarmH.ListAudioFiles)
		va.POST("/audio-files", voiceAlarmH.UploadAudioFile)
		va.DELETE("/audio-files/:name", voiceAlarmH.DeleteAudioFile)
	}

	// Serve snapshot images
	r.Static("/snapshots", "/home/hzhy/ai-monitor-service/snapshots")

	log.Printf("AI Monitor Backend listening on %s", config.Port)
	if err := r.Run(config.Port); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
