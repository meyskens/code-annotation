package handler

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/src-d/code-annotation/server/model"
	"github.com/src-d/code-annotation/server/repository"
	"github.com/src-d/code-annotation/server/serializer"
	"github.com/src-d/code-annotation/server/service"
)

// GetExperimentDetails returns a function that returns a *serializer.Response
// with the details of a requested experiment
func GetExperimentDetails(repo *repository.Experiments, assignmentsRepo *repository.Assignments) RequestProcessFunc {
	return func(r *http.Request) (*serializer.Response, error) {
		userID, err := service.GetUserID(r.Context())
		if err != nil {
			return nil, err
		}

		experimentID, err := urlParamInt(r, "experimentId")
		if err != nil {
			return nil, err
		}

		experiment, err := repo.GetByID(experimentID)
		if err != nil {
			return nil, err
		}

		if experiment == nil {
			return nil, serializer.NewHTTPError(http.StatusNotFound, "no experiment found")
		}

		progress, err := experimentProgress(assignmentsRepo, experiment.ID, userID)
		if err != nil {
			return nil, err
		}

		return serializer.NewExperimentResponse(experiment, progress), nil
	}
}

// GetExperiments returns a function that returns a *serializer.Response
// with the list of existing experiments
func GetExperiments(repo *repository.Experiments, assignmentsRepo *repository.Assignments) RequestProcessFunc {
	return func(r *http.Request) (*serializer.Response, error) {
		userID, err := service.GetUserID(r.Context())
		if err != nil {
			return nil, err
		}

		experiments, err := repo.GetAll()
		if err != nil {
			return nil, err
		}

		var progresses []float32
		for _, e := range experiments {
			progress, err := experimentProgress(assignmentsRepo, e.ID, userID)
			if err != nil {
				return nil, err
			}
			progresses = append(progresses, progress)
		}

		return serializer.NewExperimentsResponse(experiments, progresses), nil
	}
}

func experimentProgress(repo *repository.Assignments, experimentID int, userID int) (float32, error) {
	countAll, err := repo.CountUserAssignment(experimentID, userID)
	if err != nil {
		return 0, fmt.Errorf("Error count of assigments from the DB: %v", err)
	}

	countComplete, err := repo.CountCompleteUserAssignment(experimentID, userID)
	if err != nil {
		return 0, fmt.Errorf("Error count of complete assigments from the DB: %v", err)
	}

	if countAll == 0 {
		return 0, nil
	}

	return 100.0 * float32(countComplete) / float32(countAll), nil
}

type createExperimentReq struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// CreateExperiment returns a function that saves the experiment as passed in the body request
func CreateExperiment(repo *repository.Experiments) RequestProcessFunc {
	return func(r *http.Request) (*serializer.Response, error) {
		var createExperimentReq createExperimentReq
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return nil, serializer.NewHTTPError(http.StatusBadRequest, err.Error())
		}

		err = json.Unmarshal(body, &createExperimentReq)
		if err != nil {
			return nil, serializer.NewHTTPError(http.StatusBadRequest, err.Error())
		}

		experiment := &model.Experiment{
			Name:        createExperimentReq.Name,
			Description: createExperimentReq.Description,
		}

		err = repo.Create(experiment)
		if err != nil {
			return nil, err
		}

		return serializer.NewExperimentResponse(experiment, 0), nil
	}
}

type updateExperimentReq struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// UpdateExperiment returns a function that updates the experiment as passed in the body request
func UpdateExperiment(repo *repository.Experiments, assignmentsRepo *repository.Assignments) RequestProcessFunc {
	return func(r *http.Request) (*serializer.Response, error) {
		userID, err := service.GetUserID(r.Context())
		if err != nil {
			return nil, err
		}

		experimentID, err := urlParamInt(r, "experimentId")
		if err != nil {
			return nil, err
		}

		experiment, err := repo.GetByID(experimentID)
		if err != nil {
			return nil, err
		}
		if experiment == nil {
			return nil, serializer.NewHTTPError(http.StatusNotFound, "no experiment found")
		}

		var updateExperimentReq updateExperimentReq
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return nil, serializer.NewHTTPError(http.StatusBadRequest, err.Error())
		}

		err = json.Unmarshal(body, &updateExperimentReq)
		if err != nil {
			return nil, serializer.NewHTTPError(http.StatusBadRequest, err.Error())
		}

		experiment.Name = updateExperimentReq.Name
		experiment.Description = updateExperimentReq.Description

		err = repo.Update(experiment)
		if err != nil {
			return nil, err
		}

		progress, err := experimentProgress(assignmentsRepo, experiment.ID, userID)
		if err != nil {
			return nil, err
		}

		return serializer.NewExperimentResponse(experiment, progress), nil
	}
}
