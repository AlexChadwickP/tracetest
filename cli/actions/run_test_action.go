package actions

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	cienvironment "github.com/cucumber/ci-environment/go"
	"github.com/joho/godotenv"
	"github.com/kubeshop/tracetest/cli/config"
	"github.com/kubeshop/tracetest/cli/file"
	"github.com/kubeshop/tracetest/cli/formatters"
	"github.com/kubeshop/tracetest/cli/openapi"
	"github.com/kubeshop/tracetest/server/model/yaml"
	"go.uber.org/zap"
)

type RunTestConfig struct {
	DefinitionFile string
	EnvID          string
	WaitForResult  bool
	JUnit          string
}

type runTestAction struct {
	config config.Config
	logger *zap.Logger
	client *openapi.APIClient
}

var _ Action[RunTestConfig] = &runTestAction{}

type runDefParams struct {
	DefinitionFile string
	EnvID          string
	WaitForResult  bool
	JunitFile      string
	Metadata       map[string]string
}

func NewRunTestAction(config config.Config, logger *zap.Logger, client *openapi.APIClient) runTestAction {
	return runTestAction{config, logger, client}
}

func (a runTestAction) Run(ctx context.Context, args RunTestConfig) error {
	if args.DefinitionFile == "" {
		return fmt.Errorf("you must specify a definition file to run a test")
	}

	if args.JUnit != "" && !args.WaitForResult {
		return fmt.Errorf("--junit option requires --wait-for-result")
	}

	a.logger.Debug(
		"Running test from definition",
		zap.String("definitionFile", args.DefinitionFile),
		zap.String("environment", args.EnvID),
		zap.Bool("waitForResults", args.WaitForResult),
		zap.String("junit", args.JUnit),
	)

	envID, err := a.processEnv(ctx, args.EnvID)
	if err != nil {
		return fmt.Errorf("could not run definition: %w", err)
	}

	params := runDefParams{
		DefinitionFile: args.DefinitionFile,
		EnvID:          envID,
		WaitForResult:  args.WaitForResult,
		JunitFile:      args.JUnit,
		Metadata:       a.getMetadata(),
	}

	err = a.runDefinition(ctx, params)
	if err != nil {
		return fmt.Errorf("could not run definition: %w", err)
	}

	return nil
}

func stringReferencesFile(path string) bool {
	return strings.HasPrefix(path, "./")
}

func (a runTestAction) processEnv(ctx context.Context, envID string) (string, error) {
	if !stringReferencesFile(envID) { //not a file, do nothing
		return envID, nil
	}

	envVars, err := godotenv.Read(envID)
	if err != nil {
		return "", fmt.Errorf(`cannot read env file "%s": %w`, envID, err)
	}

	values := make([]openapi.EnvironmentValue, 0, len(envVars))
	for k, v := range envVars {
		values = append(values, openapi.EnvironmentValue{
			Key:   openapi.PtrString(k),
			Value: openapi.PtrString(v),
		})
	}

	name := filepath.Base(envID)

	req := openapi.Environment{
		Id:     &name,
		Name:   &name,
		Values: values,
	}

	body, resp, err := a.client.ApiApi.
		CreateEnvironment(ctx).
		Environment(req).
		Execute()
	if err != nil {
		if resp.StatusCode == http.StatusBadRequest {
			return a.updateEnv(ctx, req)
		}

		return "", fmt.Errorf("could not create environment: %w", err)
	}

	return body.GetId(), nil
}

func (a runTestAction) updateEnv(ctx context.Context, req openapi.Environment) (string, error) {
	resp, err := a.client.ApiApi.
		UpdateEnvironment(ctx, req.GetId()).
		Environment(req).
		Execute()
	if err != nil {
		return "", fmt.Errorf("could not update environment: %w", err)
	}

	if resp.StatusCode != http.StatusNoContent {
		return "", fmt.Errorf("error updating environment")
	}

	return req.GetId(), nil
}

func (a runTestAction) testFileToID(ctx context.Context, originalPath, filePath string) (string, error) {
	path := filepath.Join(originalPath, filePath)
	f, err := file.Read(path)
	if err != nil {
		return "", err
	}

	body, _, err := a.client.ApiApi.
		UpsertDefinition(ctx).
		TextDefinition(openapi.TextDefinition{
			Content: openapi.PtrString(f.Contents()),
		}).
		Execute()

	if err != nil {
		return "", fmt.Errorf("could not upsert definition: %w", err)
	}

	return body.GetId(), nil
}

func (a runTestAction) runDefinition(ctx context.Context, params runDefParams) error {
	f, err := file.Read(params.DefinitionFile)
	if err != nil {
		return err
	}

	defFile := f.Definition()
	if err = defFile.Validate(); err != nil {
		return fmt.Errorf("invalid definition file: %w", err)
	}

	return a.runDefinitionFile(ctx, f, params)
}

func (a runTestAction) runDefinitionFile(ctx context.Context, f file.File, params runDefParams) error {
	f, err := f.ResolveVariables()
	if err != nil {
		return err
	}

	if t, err := f.Definition().Transaction(); err == nil {
		for i, step := range t.Steps {
			if !stringReferencesFile(step) {
				// not referencing a file, keep the value
				continue
			}

			// references a file, resolve to its ID
			id, err := a.testFileToID(ctx, f.AbsDir(), step)
			if err != nil {
				return fmt.Errorf(`cannot transalte path "%s" to an ID: %w`, step, err)
			}

			t.Steps[i] = id
		}

		def := yaml.File{
			Type: yaml.FileTypeTransaction,
			Spec: t,
		}

		updated, err := def.Encode()
		if err != nil {
			return fmt.Errorf(`cannot encode updated transaction: %w`, err)
		}

		f, err = file.New(f.Path(), updated)
		if err != nil {
			return fmt.Errorf(`cannot recreate updated file: %w`, err)
		}
	}

	body, resp, err := a.client.ApiApi.
		ExecuteDefinition(ctx).
		TextDefinition(openapi.TextDefinition{
			Content: openapi.PtrString(f.Contents()),
			RunInformation: &openapi.RunInformation{
				Metadata:      params.Metadata,
				EnvironmentId: &params.EnvID,
			},
		}).
		Execute()

	if err != nil {
		return fmt.Errorf("could not execute definition: %w", err)
	}

	if resp.StatusCode == http.StatusCreated && !f.HasID() {
		f, err = f.SetID(body.GetId())
		if err != nil {
			return fmt.Errorf("could not update definition file: %w", err)
		}

		_, err = f.Write()
		if err != nil {
			return fmt.Errorf("could not update definition file: %w", err)
		}
	}

	runID := body.GetRunId()
	a.logger.Debug(
		"executed",
		zap.String("runID", runID),
		zap.String("runType", body.GetType()),
	)

	switch body.GetType() {
	case "Test":
		test, err := a.getTest(ctx, body.GetId())
		if err != nil {
			return fmt.Errorf("could not get test info: %w", err)
		}
		return a.testRun(ctx, test, runID, params)
	case "Transaction":
		test, err := a.getTransaction(ctx, body.GetId())
		if err != nil {
			return fmt.Errorf("could not get test info: %w", err)
		}
		return a.transactionRun(ctx, test, runID, params)
	}

	return fmt.Errorf(`unsuported run type "%s"`, body.GetType())
}

func (a runTestAction) getTransaction(ctx context.Context, id string) (openapi.Transaction, error) {
	test, _, err := a.client.ApiApi.
		GetTransaction(ctx, id).
		Execute()
	if err != nil {
		return openapi.Transaction{}, fmt.Errorf("could not execute request: %w", err)
	}

	return *test, nil
}

func (a runTestAction) getTest(ctx context.Context, id string) (openapi.Test, error) {
	test, _, err := a.client.ApiApi.
		GetTest(ctx, id).
		Execute()
	if err != nil {
		return openapi.Test{}, fmt.Errorf("could not execute request: %w", err)
	}

	return *test, nil
}

func (a runTestAction) testRun(ctx context.Context, test openapi.Test, runID string, params runDefParams) error {
	a.logger.Debug("run test", zap.Bool("wait-for-results", params.WaitForResult))
	testID := test.GetId()
	testRun, err := a.getTestRun(ctx, testID, runID)
	if err != nil {
		return fmt.Errorf("could not run test: %w", err)
	}

	if params.WaitForResult {
		updatedTestRun, err := a.waitForTestResult(ctx, testID, testRun.GetId())
		if err != nil {
			return fmt.Errorf("could not wait for result: %w", err)
		}

		testRun = updatedTestRun

		if err := a.saveJUnitFile(ctx, testID, testRun.GetId(), params.JunitFile); err != nil {
			return fmt.Errorf("could not save junit file: %w", err)
		}
	}

	tro := formatters.TestRunOutput{
		HasResults: params.WaitForResult,
		Test:       test,
		Run:        testRun,
	}

	formatter := formatters.TestRun(a.config, true)
	formattedOutput := formatter.Format(tro)
	fmt.Print(formattedOutput)

	allPassed := tro.Run.Result.GetAllPassed()
	if params.WaitForResult && !allPassed {
		// It failed, so we have to return an error status
		os.Exit(1)
	}

	return nil
}

func (a runTestAction) transactionRun(ctx context.Context, transaction openapi.Transaction, runID string, params runDefParams) error {
	rid, _ := strconv.Atoi(runID)
	a.logger.Debug("run transaction", zap.Bool("wait-for-results", params.WaitForResult))
	transactionID := transaction.GetId()
	transactionRun, err := a.getTransactionRun(ctx, transactionID, int32(rid))
	if err != nil {
		return fmt.Errorf("could not run transaction: %w", err)
	}

	if params.WaitForResult {
		updatedTestRun, err := a.waitForTransactionResult(ctx, transactionID, transactionRun.GetId())
		if err != nil {
			return fmt.Errorf("could not wait for result: %w", err)
		}

		transactionRun = updatedTestRun
	}

	tro := formatters.TransactionRunOutput{
		HasResults:  params.WaitForResult,
		Transaction: transaction,
		Run:         transactionRun,
	}

	formatter := formatters.TransactionRun(a.config, true)
	formattedOutput := formatter.Format(tro)
	fmt.Print(formattedOutput)

	if params.WaitForResult {
		if tro.Run.GetState() == "FAILED" {
			// It failed, so we have to return an error status
			os.Exit(1)
		}

		for _, step := range tro.Run.Steps {
			if !step.Result.GetAllPassed() {
				// if any test doesn't pass, fail the transaction execution
				os.Exit(1)
			}
		}
	}

	return nil
}

func (a runTestAction) saveJUnitFile(ctx context.Context, testId, testRunId, outputFile string) error {
	if outputFile == "" {
		return nil
	}

	req := a.client.ApiApi.GetRunResultJUnit(ctx, testId, testRunId)
	junit, _, err := a.client.ApiApi.GetRunResultJUnitExecute(req)
	if err != nil {
		return fmt.Errorf("could not execute request: %w", err)
	}

	f, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("could not create junit output file: %w", err)
	}

	_, err = f.WriteString(junit)

	return err

}

func (a runTestAction) getTestRun(ctx context.Context, testID, runID string) (openapi.TestRun, error) {
	run, _, err := a.client.ApiApi.
		GetTestRun(ctx, testID, runID).
		Execute()
	if err != nil {
		return openapi.TestRun{}, fmt.Errorf("could not execute request: %w", err)
	}

	return *run, nil
}

func (a runTestAction) getTransactionRun(ctx context.Context, transactionID string, runID int32) (openapi.TransactionRun, error) {
	run, _, err := a.client.ApiApi.
		GetTransactionRun(ctx, transactionID, runID).
		Execute()
	if err != nil {
		return openapi.TransactionRun{}, fmt.Errorf("could not execute request: %w", err)
	}

	return *run, nil
}

func (a runTestAction) waitForTestResult(ctx context.Context, testID, testRunID string) (openapi.TestRun, error) {
	var (
		testRun   openapi.TestRun
		lastError error
		wg        sync.WaitGroup
	)
	wg.Add(1)
	ticker := time.NewTicker(1 * time.Second) // TODO: change to websockets
	go func() {
		for {
			select {
			case <-ticker.C:
				readyTestRun, err := a.isTestReady(ctx, testID, testRunID)
				if err != nil {
					lastError = err
					wg.Done()
					return
				}

				if readyTestRun != nil {
					testRun = *readyTestRun
					wg.Done()
					return
				}
			}
		}
	}()
	wg.Wait()

	if lastError != nil {
		return openapi.TestRun{}, lastError
	}

	return testRun, nil
}

func (a runTestAction) waitForTransactionResult(ctx context.Context, transactionID, transactionRunID string) (openapi.TransactionRun, error) {
	var (
		transactionRun openapi.TransactionRun
		lastError      error
		wg             sync.WaitGroup
	)
	wg.Add(1)
	ticker := time.NewTicker(1 * time.Second) // TODO: change to websockets
	go func() {
		for {
			select {
			case <-ticker.C:
				readyTransactionRun, err := a.isTransactionReady(ctx, transactionID, transactionRunID)
				if err != nil {
					lastError = err
					wg.Done()
					return
				}

				if readyTransactionRun != nil {
					transactionRun = *readyTransactionRun
					wg.Done()
					return
				}
			}
		}
	}()
	wg.Wait()

	if lastError != nil {
		return openapi.TransactionRun{}, lastError
	}

	return transactionRun, nil
}

func (a runTestAction) isTestReady(ctx context.Context, testID, testRunId string) (*openapi.TestRun, error) {
	req := a.client.ApiApi.GetTestRun(ctx, testID, testRunId)
	run, _, err := a.client.ApiApi.GetTestRunExecute(req)
	if err != nil {
		return &openapi.TestRun{}, fmt.Errorf("could not execute GetTestRun request: %w", err)
	}

	if *run.State == "FAILED" || *run.State == "FINISHED" {
		return run, nil
	}

	return nil, nil
}

func (a runTestAction) isTransactionReady(ctx context.Context, transactionID, transactionRunId string) (*openapi.TransactionRun, error) {
	runId, err := strconv.Atoi(transactionRunId)
	if err != nil {
		return nil, fmt.Errorf("invalid transaction run id format: %w", err)
	}

	req := a.client.ApiApi.GetTransactionRun(ctx, transactionID, int32(runId))
	run, _, err := a.client.ApiApi.GetTransactionRunExecute(req)
	if err != nil {
		return nil, fmt.Errorf("could not execute GetTestRun request: %w", err)
	}

	if *run.State == "FAILED" || *run.State == "FINISHED" {
		return run, nil
	}

	return nil, nil
}

func (a runTestAction) getMetadata() map[string]string {
	ci := cienvironment.DetectCIEnvironment()
	if ci == nil {
		return map[string]string{}
	}

	metadata := map[string]string{
		"name":        ci.Name,
		"url":         ci.URL,
		"buildNumber": ci.BuildNumber,
	}

	if ci.Git != nil {
		metadata["branch"] = ci.Git.Branch
		metadata["tag"] = ci.Git.Tag
		metadata["revision"] = ci.Git.Revision
	}

	return metadata
}
