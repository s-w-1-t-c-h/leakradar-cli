package api

import "context"

// AsyncTaskResult is the response of a .../unlock/task endpoint. The API
// documents this only as an arbitrary JSON object (no fixed schema), but in
// practice carries a "task_id" usable with GetTaskStatus.
type AsyncTaskResult map[string]interface{}

// TaskID extracts the task_id field, if present.
func (r AsyncTaskResult) TaskID() string {
	if v, ok := r["task_id"].(string); ok {
		return v
	}
	return ""
}

// GetTaskStatus calls GET /tasks/{task_id} to poll a background task queued
// by one of the .../unlock/task endpoints.
func (c *Client) GetTaskStatus(ctx context.Context, taskID string) (*TaskStatus, error) {
	var out TaskStatus
	err := c.doJSON(ctx, requestOpts{
		method: "GET",
		path:   "/tasks/" + pathEscape(taskID),
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}
