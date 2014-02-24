// Copyright 2013 Andreas Koch. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package itemhandler

import (
	"fmt"
	"github.com/andreaskoch/allmark2/common/config"
	"github.com/andreaskoch/allmark2/common/index"
	"github.com/andreaskoch/allmark2/common/logger"
	"github.com/andreaskoch/allmark2/common/paths"
	"github.com/andreaskoch/allmark2/common/route"
	"github.com/andreaskoch/allmark2/services/conversion"
	"github.com/andreaskoch/allmark2/ui/web/orchestrator"
	"github.com/andreaskoch/allmark2/ui/web/server/handler/handlerutil"
	"github.com/andreaskoch/allmark2/ui/web/view/templates"
	"github.com/andreaskoch/allmark2/ui/web/view/viewmodel"
	"io"
	"net/http"
)

func New(logger logger.Logger, config *config.Config, itemIndex *index.ItemIndex, fileIndex *index.FileIndex, patherFactory paths.PatherFactory, converter conversion.Converter) *ItemHandler {

	viewModelOrchestrator := orchestrator.NewViewModelOrchestrator(itemIndex, converter)

	return &ItemHandler{
		logger:                logger,
		itemIndex:             itemIndex,
		fileIndex:             fileIndex,
		config:                config,
		patherFactory:         patherFactory,
		viewModelOrchestrator: viewModelOrchestrator,
	}
}

type ItemHandler struct {
	logger                logger.Logger
	itemIndex             *index.ItemIndex
	fileIndex             *index.FileIndex
	config                *config.Config
	patherFactory         paths.PatherFactory
	viewModelOrchestrator orchestrator.ViewModelOrchestrator
}

func (handler *ItemHandler) Func() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {

		// get the request route
		requestRoute, err := handlerutil.GetRouteFromRequest(r)
		if err != nil {
			fmt.Fprintln(w, "%s", err)
			return
		}

		// make sure the request body is closed
		defer r.Body.Close()

		// stage 1: check for theme files
		themeRoute, err := route.NewFromRequest("theme")
		if err != nil {
			fmt.Fprintln(w, "%s", err)
			return
		}

		if isThemeFile := requestRoute.IsChildOf(themeRoute); isThemeFile {

			if file, found := handler.fileIndex.IsMatch(*requestRoute); found {
				fileContentProvider := file.ContentProvider()
				data, err := fileContentProvider.Data()
				if err != nil {
					return
				}

				fmt.Fprintf(w, "%s", data)
				return
			}
		}

		// stage 2: check if there is a item for the request
		if item, found := handler.itemIndex.IsMatch(*requestRoute); found {

			// create the view model
			pathProvider := handler.patherFactory.Relative(item.Route())
			viewModel := handler.viewModelOrchestrator.GetViewModel(pathProvider, item)

			// render the view model
			render(w, viewModel)
			return
		}

		// stage 3: check if there is a file for the request
		if file, found := handler.itemIndex.IsFileMatch(*requestRoute); found {
			contentProvider := file.ContentProvider()

			// read the file data
			data, err := contentProvider.Data()
			if err != nil {
				return
			}

			fmt.Fprintf(w, "%s", data)
			return
		}

		fmt.Fprintln(w, fmt.Sprintf("item %q not found.", requestRoute))
		return
	}
}

func render(writer io.Writer, viewModel viewmodel.Model) {

	templateProvider := templates.NewProvider(".")

	// get a template
	if template, err := templateProvider.GetFullTemplate(viewModel.Type); err == nil {

		err := template.Execute(writer, viewModel)
		if err != nil {
			fmt.Println(err)
		}

	} else {

		fmt.Fprintf(writer, "No template for item of type %q.", viewModel.Type)

	}

}
