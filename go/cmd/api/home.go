package main

import (
	"net/http"

	jsonapi "nathejk.dk/cmd/api/app"
)

func (app *application) homeHandler(w http.ResponseWriter, r *http.Request) {
	config := map[string]any{
		"timeCountdown": app.config.countdown.time,
		"videos":        app.config.countdown.videos,
	}
	err := app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"config": config}, nil)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

/*
func (app *application) showProductHandler(w http.ResponseWriter, r *http.Request) {
	id, err := app.ReadIDParam(r)
	if err != nil {
		app.NotFoundResponse(w, r)
		return
	}
	product, err := app.models.Products.Get(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.NotFoundResponse(w, r)
		default:
			app.ServerErrorResponse(w, r, err)
		}
		return
	}
	product.TargetGroup.Filters = []data.Filter{
		{Slug: "gender", Label: "Køn", Type: data.FilterTypeRadio, Options: []data.FilterOption{
			{Label: "Mand", Value: "M"},
			{Label: "Kvinde", Value: "F"},
			{Label: "Ukendt", Value: "X"},
		}},
	}
	err = app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"product": product}, nil)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

func (app *application) createProductHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Title          string        `json:"title"`
		Summary        string        `json:"summary"`
		Description    string        `json:"description"`
		TypeID         int64         `json:"type_id"`
		TypeLabel      string        `json:"type_label"`
		Baseprice      int32         `json:"baseprice"`
		Currency       data.Currency `json:"currency"`
		HasCoupons     bool          `json:"has_coupons"`
		AutomatedStudy bool          `json:"automated_study"`
		Stimuli        struct {
			Type     data.StimuliType `json:"type"`
			MinCount int16            `json:"min_count"`
			MaxCount int16            `json:"max_count"`
		} `json:"stimuli"`
	}

	if err := app.ReadJSON(w, r, &input); err != nil {
		app.BadRequestResponse(w, r, err)
		return
	}

	product := &data.Product{
		Title:          input.Title,
		Summary:        input.Summary,
		Description:    input.Description,
		TypeID:         input.TypeID,
		TypeLabel:      input.TypeLabel,
		Baseprice:      input.Baseprice,
		Currency:       input.Currency,
		HasCoupons:     input.HasCoupons,
		AutomatedStudy: input.AutomatedStudy,
		Stimuli: data.Stimuli{
			Type:     input.Stimuli.Type,
			MinCount: input.Stimuli.MinCount,
			MaxCount: input.Stimuli.MaxCount,
		},
	}

	v := validator.New()

	if product.Validate(v); !v.Valid() {
		app.FailedValidationResponse(w, r, v.Errors)
		return
	}

	if err := app.models.Products.Insert(product); err != nil {
		app.ServerErrorResponse(w, r, err)
		return
	}

	headers := make(http.Header)
	headers.Set("Location", fmt.Sprintf("/v1/products/%d", product.ID))

	err := app.WriteJSON(w, http.StatusCreated, jsonapi.Envelope{"product": product}, headers)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

func (app *application) updateProductHandler(w http.ResponseWriter, r *http.Request) {
	id, err := app.ReadIDParam(r)
	if err != nil {
		app.NotFoundResponse(w, r)
		return
	}
	product, err := app.models.Products.Get(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.NotFoundResponse(w, r)
		default:
			app.ServerErrorResponse(w, r, err)
		}
		return
	}
	// If the request contains a X-Expected-Version header, verify that the
	// version in the database matches the expected version specified in the header.
	if r.Header.Get("X-Expected-Version") != "" {
		if strconv.FormatInt(int64(product.Version), 32) != r.Header.Get("X-Expected-Version") {
			app.EditConflictResponse(w, r)
			return
		}
	}

	var input struct {
		Title          *string        `json:"title"`
		Summary        *string        `json:"summary"`
		Description    *string        `json:"description"`
		TypeID         *int64         `json:"type_id"`
		TypeLabel      *string        `json:"type_label"`
		Baseprice      *int32         `json:"baseprice"`
		Currency       *data.Currency `json:"currency"`
		HasCoupons     *bool          `json:"has_coupons"`
		AutomatedStudy *bool          `json:"automated_study"`
		Stimuli        *struct {
			Type     *data.StimuliType `json:"type"`
			MinCount *int16            `json:"min_count"`
			MaxCount *int16            `json:"max_count"`
		} `json:"stimuli"`
	}
	if err = app.ReadJSON(w, r, &input); err != nil {
		app.BadRequestResponse(w, r, err)
		return
	}
	if input.Title != nil {
		product.Title = *input.Title
	}
	if input.Description != nil {
		product.Description = *input.Description
	}
	if input.Summary != nil {
		product.Summary = *input.Summary
	}
	if input.TypeID != nil {
		product.TypeID = *input.TypeID
	}
	if input.TypeLabel != nil {
		product.TypeLabel = *input.TypeLabel
	}
	if input.Baseprice != nil {
		product.Baseprice = *input.Baseprice
	}
	if input.Currency != nil {
		product.Currency = *input.Currency
	}
	if input.HasCoupons != nil {
		product.HasCoupons = *input.HasCoupons
	}
	if input.AutomatedStudy != nil {
		product.AutomatedStudy = *input.AutomatedStudy
	}
	if input.Stimuli != nil {
		stimuli := *input.Stimuli
		if stimuli.Type != nil {
			product.Stimuli.Type = *stimuli.Type
		}
		if stimuli.MinCount != nil {
			product.Stimuli.MinCount = *stimuli.MinCount
		}
		if stimuli.MaxCount != nil {
			product.Stimuli.MaxCount = *stimuli.MaxCount
		}
	}
	v := validator.New()
	if product.Validate(v); !v.Valid() {
		app.FailedValidationResponse(w, r, v.Errors)
		return
	}
	err = app.models.Products.Update(product)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrEditConflict):
			app.EditConflictResponse(w, r)
		default:
			app.ServerErrorResponse(w, r, err)
		}
		return
	}
	err = app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"product": product}, nil)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

func (app *application) deleteProductHandler(w http.ResponseWriter, r *http.Request) {
	id, err := app.ReadIDParam(r)
	if err != nil {
		app.NotFoundResponse(w, r)
		return
	}
	err = app.models.Products.Delete(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.NotFoundResponse(w, r)
		default:
			app.ServerErrorResponse(w, r, err)
		}
		return
	}

	err = app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"message": "product successfully deleted"}, nil)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}

func (app *application) listProductsHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Title  string
		Genres []string
		data.Filters
	}
	v := validator.New()

	qs := r.URL.Query()

	input.Title = app.ReadString(qs, "title", "")
	input.Genres = app.ReadCSV(qs, "genres", []string{})

	input.Filters.Page = app.ReadInt(qs, "page", 1, v)
	input.Filters.PageSize = app.ReadInt(qs, "page_size", 20, v)
	input.Filters.Sort = app.ReadString(qs, "sort", "id")
	input.Filters.SortSafelist = []string{"id", "title", "year", "runtime", "-id", "-title", "-year", "-runtime"}

	if input.Filters.Validate(v); !v.Valid() {
		app.FailedValidationResponse(w, r, v.Errors)
		return
	}

	products, metadata, err := app.models.Products.GetAll(input.Title, input.Genres, input.Filters)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
		return
	}

	err = app.WriteJSON(w, http.StatusOK, jsonapi.Envelope{"metadata": metadata, "products": products}, nil)
	if err != nil {
		app.ServerErrorResponse(w, r, err)
	}
}
*/
