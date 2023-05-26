//go:build ignore

package main

import (
	"context"

	"github.com/pandodao/PAL9000/service"
)

var PluginInstance = Plugin{}

type Plugin struct{}

func (p Plugin) PluginName() string {
	return "example"
}

func (p Plugin) ExecuteBefore(ctx context.Context, req service.PluginBeforeExecuteRequest) (*service.PluginBeforeExecuteResponse, error) {
	resp := &service.PluginBeforeExecuteResponse{
		TerminatePlugins: true,
		TerminateRequest: true,
		CustomResponse:   []string{"example1", "example2"},
	}

	return resp, nil
}

func (p Plugin) ExecuteAfter(ctx context.Context, req service.PluginAfterExecuteRequest) (*service.PluginAfterExecuteResponse, error) {
	return &service.PluginAfterExecuteResponse{
		ModifiedResponse: "modified_response",
		TerminatePlugins: true,
	}, nil
}
