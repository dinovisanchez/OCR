package router

import (
	"ia-extraction/config"
	"ia-extraction/handlers/controller"
	"ia-extraction/provider/postgres"
	"ia-extraction/repositories"
	"ia-extraction/services"
	"io/ioutil"
	"log"

	ginCommons "github.com/biaenergy/bia-commons-go/gin"
	"github.com/gin-gonic/gin"
)


func NewRouter() {
	db := postgres.NewPostgresDB()
	gin.SetMode(gin.ReleaseMode)
	gin.DefaultWriter = ioutil.Discard

	prefix := "/ms-ia-extraction"

	biaRouterConfig := ginCommons.BiaRouterConfig{
		Production:           config.ProdEnvironment,
		DiscardDefaultWriter: false,
		PrefixPath:           prefix,
		SkipPaths:            []string{},
		SkipPathsAPM:         []string{},
	}

	router, apiGroup, _, _ := ginCommons.CreateRouter(biaRouterConfig)

	gormDB := db.GetConnection()
	sqlDB, _ := gormDB.DB()
	ExtractionRepository := repositories.NewPostgresExtractionRepository(sqlDB)
	DocumentTypePromptRepository := repositories.NewPostgresDocumentTypePromptRepository(gormDB)
	GeminiClient := services.NewGeminiClient()
	ExtractionService := services.NewExtractionService(ExtractionRepository, DocumentTypePromptRepository, GeminiClient)
	ExtractionController := controller.NewExtractionController(ExtractionService)
	DocumentTypePromptController := controller.NewDocumentTypePromptController(DocumentTypePromptRepository)

	v1 := router.Group(prefix)
	{
		extraccion := v1.Group("/extraction_information")
		{
			extraccion.POST("", ExtractionController.Extract)
		}

		prompts := v1.Group("/document_type_prompts")
		{
			prompts.GET("", DocumentTypePromptController.List)
			prompts.GET("/:document_type", DocumentTypePromptController.GetByDocumentType)
			prompts.POST("", DocumentTypePromptController.Create)
			prompts.PUT("/:id", DocumentTypePromptController.Update)
			prompts.DELETE("/:id", DocumentTypePromptController.Delete)
		}
	}

	_ = apiGroup

	err := router.Run(":8080")
	if err != nil {
		log.Fatal(err.Error())
	}
}