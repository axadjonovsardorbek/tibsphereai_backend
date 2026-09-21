// Package docs - swag init generatsiya qilishi kerak bo'lgan joy (Taskfile.yml: task swag-init).
// Hozircha minimal stub: /api/swagger/index.html ochiladi, lekin endpointlar
// ro'yxati bo'sh. `swag init -g api/api.go -o api/docs` ishga tushirilgach
// ushbu fayl to'liq generatsiya qilingan versiya bilan almashtiriladi.
package docs

import "github.com/swaggo/swag"

const docTemplate = `{
    "schemes": {{ marshal .Schemes }},
    "swagger": "2.0",
    "info": {
        "description": "{{escape .Description}}",
        "title": "{{.Title}}",
        "contact": {},
        "version": "{{.Version}}"
    },
    "host": "{{.Host}}",
    "basePath": "{{.BasePath}}",
    "paths": {},
    "securityDefinitions": {
        "BearerAuth": {
            "type": "apiKey",
            "name": "Authorization",
            "in": "header"
        }
    }
}`

// SwaggerInfo holds exported Swagger Info so clients can modify it
var SwaggerInfo = &swag.Spec{
	Version:          "1.0",
	Host:             "",
	BasePath:         "/",
	Schemes:          []string{},
	Title:            "MedicAI API",
	Description:      "API for MedicAI clinical simulation platform",
	InfoInstanceName: "swagger",
	SwaggerTemplate:  docTemplate,
	LeftDelim:        "{{",
	RightDelim:       "}}",
}

func init() {
	swag.Register(SwaggerInfo.InstanceName(), SwaggerInfo)
}
