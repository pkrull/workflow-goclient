package activity

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/3dsim/workflow-goclient/models"
	"github.com/3dsim/workflow-goclient/workflow/workflowfakes"
	"github.com/go-openapi/swag"
	log "github.com/inconshreveable/log15"
	"github.com/stretchr/testify/assert"
)

var logger log.Logger

func init() {
	logger = log.New()
	logger.SetHandler(log.LvlFilterHandler(log.LvlDebug, log.CallerFileHandler(log.StdoutHandler)))
}

func TestDoExpectsCompleteFailedActivityCalledWhenErrorOccurs(t *testing.T) {
	// arrange
	fakeWorkflowClient := &workflowfakes.FakeClient{}
	worker := &Worker{WorkflowClient: fakeWorkflowClient, Logger: logger}
	activityID := "activity id"
	workflowID := "workflow id"
	taskToken := "token"
	errorReason := "Some error"

	// act
	worker.Do(context.Background(), workflowID, activityID, taskToken, func(context.Context, chan<- int) (interface{}, error) {
		return nil, errors.New(errorReason)
	})

	// assert
	assert.Equal(t, 1, fakeWorkflowClient.CompleteFailedActivityCallCount(), "Expected to call CompleteFailedActivity once")
	actualWorkflowID, actualActivityID, actualErrorReason, actualErrorDetails := fakeWorkflowClient.CompleteFailedActivityArgsForCall(0)
	assert.Equal(t, workflowID, actualWorkflowID, "Expected workflow ID passed to CompleteFailedActivity")
	assert.Equal(t, activityID, actualActivityID, "Expected activity ID passed to CompleteFailedActivity")
	assert.Equal(t, errorReason, actualErrorReason, "Expected error reason passed to CompleteFailedActivity")
	assert.Equal(t, "", actualErrorDetails, "Expected error details passed to CompleteFailedActivity")
}

func TestDoExpectsCompleteSuccessfulActivityCalledWhenNoErrorOccurs(t *testing.T) {
	// arrange
	fakeWorkflowClient := &workflowfakes.FakeClient{}
	worker := &Worker{WorkflowClient: fakeWorkflowClient, Logger: logger}
	activityID := "activity id"
	workflowID := "workflow id"
	taskToken := "token"
	result := struct{ SomeField string }{"the result"}

	// act
	worker.Do(context.Background(), workflowID, activityID, taskToken, func(context.Context, chan<- int) (interface{}, error) {
		return result, nil
	})

	// assert
	assert.Equal(t, 1, fakeWorkflowClient.CompleteSuccessfulActivityCallCount(), "Expected to call CompleteSuccessfulActivity once")
	actualWorkflowID, actualActivityID, actualResult := fakeWorkflowClient.CompleteSuccessfulActivityArgsForCall(0)
	assert.Equal(t, workflowID, actualWorkflowID, "Expected workflow ID passed to CompleteSuccessfulActivity")
	assert.Equal(t, activityID, actualActivityID, "Expected activity ID passed to CompleteSuccessfulActivity")
	assert.Equal(t, result, actualResult, "Expected result passed to CompleteSuccessfulActivity")
}

func TestDoExpectsHeartbeatActivityWithTokenCalled(t *testing.T) {
	// arrange
	fakeWorkflowClient := &workflowfakes.FakeClient{}
	worker := &Worker{WorkflowClient: fakeWorkflowClient, HeartbeatInterval: 7 * time.Millisecond, Logger: logger}
	activityID := "activity id"
	workflowID := "workflow id"
	taskToken := "token"

	// act
	worker.Do(context.Background(), workflowID, activityID, taskToken, func(context.Context, chan<- int) (interface{}, error) {
		// Wait a little time for heartbeat
		time.Sleep(10 * time.Millisecond)
		return nil, nil
	})

	// assert
	assert.Equal(t, 1, fakeWorkflowClient.HeartbeatActivityWithTokenCallCount(), "Expected to call HeartbeatActivityWithToken once")
	actualTaskToken, actualActivityID, actualDetails := fakeWorkflowClient.HeartbeatActivityWithTokenArgsForCall(0)
	assert.Equal(t, taskToken, actualTaskToken, "Expected task token passed to HeartbeatActivityWithToken")
	assert.Equal(t, activityID, actualActivityID, "Expected activityID passed to HeartbeatActivityWithToken")
	assert.NotEmpty(t, taskToken, actualDetails, "Expected details passed to HeartbeatActivityWithToken to not be empty")
}

func TestDoWhenCancellationRequestedExpectsCompleteCancelledActivityCalled(t *testing.T) {
	// arrange

	// This test sets up a worker that will heartbeat every 7 ms.  The heartbeat response is mocked out to return
	// cancelled = true.  At that point the worker should close the context that was passed to the worker function and
	// the worker function will return.  The worker function is setup to timeout after 30 ms if nothing has happened
	// by then.
	fakeWorkflowClient := &workflowfakes.FakeClient{}
	worker := &Worker{WorkflowClient: fakeWorkflowClient, HeartbeatInterval: 7 * time.Millisecond, Logger: logger}
	activityID := "activity id"
	workflowID := "workflow id"
	taskToken := "token"

	heartbeatToReturn := &models.Heartbeat{
		TaskToken:  swag.String(taskToken),
		ActivityID: swag.String(activityID),
		Cancelled:  true,
	}
	fakeWorkflowClient.HeartbeatActivityWithTokenReturns(heartbeatToReturn, nil)

	// act
	worker.Do(context.Background(), workflowID, activityID, taskToken, func(ctx context.Context, percentCompleteChan chan<- int) (interface{}, error) {
		select {
		case <-ctx.Done():
		case <-time.After(30 * time.Millisecond):
			t.Error("Did not receive the cancellation in time")
		}
		return nil, nil
	})

	// assert
	assert.True(t, fakeWorkflowClient.HeartbeatActivityWithTokenCallCount() >= 1, "Expected to call HeartbeatActivityWithToken at least once")
	actualTaskToken, actualActivityID, actualDetails := fakeWorkflowClient.HeartbeatActivityWithTokenArgsForCall(0)
	assert.Equal(t, taskToken, actualTaskToken, "Expected task token passed to HeartbeatActivityWithToken")
	assert.Equal(t, activityID, actualActivityID, "Expected activityID passed to HeartbeatActivityWithToken")
	assert.NotEmpty(t, taskToken, actualDetails, "Expected details passed to HeartbeatActivityWithToken to not be empty")
	assert.Equal(t, 1, fakeWorkflowClient.CompleteCancelledActivityCallCount(), "Expected to call CompleteCancelledActivity once")
	actualWorkflowID, actualActivityID, actualReason, actualDetails := fakeWorkflowClient.CompleteCancelledActivityArgsForCall(0)
	assert.Equal(t, workflowID, actualWorkflowID, "Expected workflow ID passed to CompleteCancelledActivity")
	assert.Equal(t, activityID, actualActivityID, "Expected activity ID passed to CompleteCancelledActivity")
	assert.Equal(t, completedMessage, actualDetails, "Expected to pass details that the work finished")
	assert.Equal(t, cancelledReason, actualReason, "Expected to pass reason for the cancellation")
}

func TestDoWhenCancellationRequestedAndFunctionErrorsExpectsCompleteCancelledActivityCalled(t *testing.T) {
	// arrange
	fakeWorkflowClient := &workflowfakes.FakeClient{}
	worker := &Worker{WorkflowClient: fakeWorkflowClient, HeartbeatInterval: 7 * time.Millisecond, Logger: logger}
	activityID := "activity id"
	workflowID := "workflow id"
	taskToken := "token"
	errMsg := "Some cancellation error"
	heartbeatToReturn := &models.Heartbeat{
		ActivityID: swag.String(activityID),
		Cancelled:  true,
	}
	fakeWorkflowClient.HeartbeatActivityWithTokenReturns(heartbeatToReturn, nil)

	// act
	worker.Do(context.Background(), workflowID, activityID, taskToken, func(ctx context.Context, percentCompleteChan chan<- int) (interface{}, error) {
		select {
		case <-ctx.Done():
		case <-time.After(30 * time.Millisecond):
			t.Error("Did not receive the cancellation in time")
		}
		return nil, errors.New(errMsg)
	})

	// assert
	assert.True(t, fakeWorkflowClient.HeartbeatActivityWithTokenCallCount() >= 1, "Expected to call HeartbeatActivityWithToken at least once")
	actualTaskToken, actualActivityID, actualDetails := fakeWorkflowClient.HeartbeatActivityWithTokenArgsForCall(0)
	assert.Equal(t, taskToken, actualTaskToken, "Expected task token passed to HeartbeatActivityWithToken")
	assert.Equal(t, activityID, actualActivityID, "Expected activityID passed to HeartbeatActivityWithToken")
	assert.NotEmpty(t, taskToken, actualDetails, "Expected details passed to HeartbeatActivityWithToken to not be empty")
	assert.Equal(t, 1, fakeWorkflowClient.CompleteCancelledActivityCallCount(), "Expected to call CompleteCancelledActivity once")
	actualWorkflowID, actualActivityID, actualReason, actualDetails := fakeWorkflowClient.CompleteCancelledActivityArgsForCall(0)
	assert.Equal(t, workflowID, actualWorkflowID, "Expected workflow ID passed to CompleteCancelledActivity")
	assert.Equal(t, activityID, actualActivityID, "Expected activity ID passed to CompleteCancelledActivity")
	assert.Equal(t, errMsg, actualDetails, "Expected to error details")
	assert.Equal(t, cancelledReason, actualReason, "Expected to pass reason for the cancellation")
}

func TestDoWhenCancellationRequestedAndFunctionBlocksForeverExpectsCompleteCancelledActivityCalledAfterTimeout(t *testing.T) {
	// arrange
	fakeWorkflowClient := &workflowfakes.FakeClient{}
	worker := &Worker{
		WorkflowClient:      fakeWorkflowClient,
		HeartbeatInterval:   7 * time.Millisecond,
		CancellationTimeout: 10 * time.Millisecond,
		Logger:              logger,
	}
	activityID := "activity id"
	workflowID := "workflow id"
	taskToken := "token"
	heartbeatToReturn := &models.Heartbeat{
		ActivityID: swag.String(activityID),
		Cancelled:  true,
	}
	fakeWorkflowClient.HeartbeatActivityWithTokenReturns(heartbeatToReturn, nil)

	// act
	worker.Do(context.Background(), workflowID, activityID, taskToken, func(ctx context.Context, percentCompleteChan chan<- int) (interface{}, error) {
		<-ctx.Done()
		time.Sleep(30 * time.Millisecond)
		return nil, errors.New("Unexpected error")
	})

	// assert
	assert.True(t, fakeWorkflowClient.HeartbeatActivityWithTokenCallCount() >= 1, "Expected to call HeartbeatActivityWithToken at least once")
	actualTaskToken, actualActivityID, actualDetails := fakeWorkflowClient.HeartbeatActivityWithTokenArgsForCall(0)
	assert.Equal(t, taskToken, actualTaskToken, "Expected task token passed to HeartbeatActivityWithToken")
	assert.Equal(t, activityID, actualActivityID, "Expected activityID passed to HeartbeatActivityWithToken")
	assert.NotEmpty(t, taskToken, actualDetails, "Expected details passed to HeartbeatActivityWithToken to not be empty")
	assert.Equal(t, 1, fakeWorkflowClient.CompleteCancelledActivityCallCount(), "Expected to call CompleteCancelledActivity once")
	actualWorkflowID, actualActivityID, actualReason, actualDetails := fakeWorkflowClient.CompleteCancelledActivityArgsForCall(0)
	assert.Equal(t, workflowID, actualWorkflowID, "Expected workflow ID passed to CompleteCancelledActivity")
	assert.Equal(t, activityID, actualActivityID, "Expected activity ID passed to CompleteCancelledActivity")
	assert.Equal(t, timeoutErrorMessage, actualDetails, "Expected to pass empty details")
	assert.Equal(t, cancelledReason, actualReason, "Expected to pass reason for the cancellation")
}

func TestDoExpectsUpdateActivityPercentCompleteCalledWhenProgressIsMade(t *testing.T) {
	// arrange
	fakeWorkflowClient := &workflowfakes.FakeClient{}
	worker := &Worker{WorkflowClient: fakeWorkflowClient, Logger: logger}
	activityID := "activity id"
	workflowID := "workflow id"
	taskToken := "token"

	// act
	worker.Do(context.Background(), workflowID, activityID, taskToken, func(ctx context.Context, percentCompleteChan chan<- int) (interface{}, error) {
		percentCompleteChan <- 30
		percentCompleteChan <- 60
		percentCompleteChan <- 100
		return nil, nil
	})

	// assert
	assert.Equal(t, 3, fakeWorkflowClient.UpdateActivityPercentCompleteCallCount(), "Expected to call UpdateActivityPercentComplete once")
	actualWorkflowID, actualActivityID, actualPercentComplete := fakeWorkflowClient.UpdateActivityPercentCompleteArgsForCall(0)
	assert.Equal(t, workflowID, actualWorkflowID, "Expected workflow ID passed to UpdateActivityPercentComplete")
	assert.Equal(t, activityID, actualActivityID, "Expected activity ID passed to UpdateActivityPercentComplete")
	assert.Equal(t, 30, actualPercentComplete, "Expected percent complete passed to UpdateActivityPercentComplete")
}

func TestDoExpectsUpdateActivityPercentCompleteCalledOnceWhenSameValuesAreSentConsecutively(t *testing.T) {
	// arrange
	fakeWorkflowClient := &workflowfakes.FakeClient{}
	worker := &Worker{WorkflowClient: fakeWorkflowClient, Logger: logger}
	activityID := "activity id"
	workflowID := "workflow id"
	taskToken := "token"

	// act
	worker.Do(context.Background(), workflowID, activityID, taskToken, func(ctx context.Context, percentCompleteChan chan<- int) (interface{}, error) {
		percentCompleteChan <- 30
		percentCompleteChan <- 30
		percentCompleteChan <- 30
		return nil, nil
	})

	// assert
	assert.Equal(t, 1, fakeWorkflowClient.UpdateActivityPercentCompleteCallCount(), "Expected to call UpdateActivityPercentComplete once")
	actualWorkflowID, actualActivityID, actualPercentComplete := fakeWorkflowClient.UpdateActivityPercentCompleteArgsForCall(0)
	assert.Equal(t, workflowID, actualWorkflowID, "Expected workflow ID passed to UpdateActivityPercentComplete")
	assert.Equal(t, activityID, actualActivityID, "Expected activity ID passed to UpdateActivityPercentComplete")
	assert.Equal(t, 30, actualPercentComplete, "Expected percent complete passed to UpdateActivityPercentComplete")
}
