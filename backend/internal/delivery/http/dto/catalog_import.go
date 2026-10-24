package dto

import "e-commerce-backend/internal/usecase"

type ImportRowErrorResponse struct {
	Line    int    `json:"line" example:"7"`
	SKU     string `json:"sku,omitempty" example:"YM-015"`
	Field   string `json:"field,omitempty" example:"price"`
	Code    string `json:"code" example:"invalid_format"`
	Message string `json:"message" example:"price must be a plain decimal number like 29.99, got \"free\""`
}

type ImportDuplicateResponse struct {
	SKU         string `json:"sku" example:"BS-021"`
	Lines       []int  `json:"lines" example:"11,56,89"`
	AppliedLine int    `json:"applied_line" example:"89"`
}

type ImportReportResponse struct {
	Mode                    string                    `json:"mode" enums:"upsert,skip,fail" example:"upsert"`
	DryRun                  bool                      `json:"dry_run" example:"false"`
	Atomic                  bool                      `json:"atomic" example:"false"`
	CreateMissingCategories bool                      `json:"create_missing_categories" example:"true"`
	Committed               bool                      `json:"committed" example:"true"`
	TotalRows               int                       `json:"total_rows" example:"97"`
	BlankRows               int                       `json:"blank_rows" example:"2"`
	Created                 int                       `json:"created" example:"81"`
	Updated                 int                       `json:"updated" example:"0"`
	Skipped                 int                       `json:"skipped" example:"0"`
	Failed                  int                       `json:"failed" example:"9"`
	CreatedCategories       []string                  `json:"created_categories"`
	Duplicates              []ImportDuplicateResponse `json:"duplicates"`
	Errors                  []ImportRowErrorResponse  `json:"errors"`
}

func NewImportReportResponse(r *usecase.ImportReport) ImportReportResponse {
	resp := ImportReportResponse{
		Mode:                    string(r.Options.Mode),
		DryRun:                  r.Options.DryRun,
		Atomic:                  r.Options.Atomic,
		CreateMissingCategories: r.Options.CreateMissingCategories,
		Committed:               r.Committed,
		TotalRows:               r.TotalRows,
		BlankRows:               r.BlankRows,
		Created:                 r.Created,
		Updated:                 r.Updated,
		Skipped:                 r.Skipped,
		Failed:                  r.Failed,
		CreatedCategories:       append([]string{}, r.CreatedCategories...),
		Duplicates:              make([]ImportDuplicateResponse, 0, len(r.Duplicates)),
		Errors:                  make([]ImportRowErrorResponse, 0, len(r.Errors)),
	}
	for _, d := range r.Duplicates {
		resp.Duplicates = append(resp.Duplicates, ImportDuplicateResponse{SKU: d.SKU, Lines: d.Lines, AppliedLine: d.AppliedLine})
	}
	for _, e := range r.Errors {
		resp.Errors = append(resp.Errors, ImportRowErrorResponse{Line: e.Line, SKU: e.SKU, Field: e.Field, Code: e.Code, Message: e.Message})
	}
	return resp
}
