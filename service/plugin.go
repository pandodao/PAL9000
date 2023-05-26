package service

import (
	"context"

	"github.com/pandodao/PAL9000/config"
	"github.com/pandodao/botastic-go"
)

type Plugin = config.Plugin

type PluginBeforeExecuteRequest Message

type PluginBeforeExecuteResponse struct {
	TerminateRequest bool     `json:"terminate_request"`
	TerminatePlugins bool     `json:"terminate_plugins"`
	ModifiedRequest  string   `json:"modified_request"`
	CustomResponse   []string `json:"custom_response"`
}

type PluginBeforeExecutor interface {
	ExecuteBefore(context.Context, PluginBeforeExecuteRequest) (*PluginBeforeExecuteResponse, error)
}

type PluginAfterExecuteRequest botastic.ConvTurn

type PluginAfterExecuteResponse struct {
	TerminatePlugins bool   `json:"terminate_plugins"`
	ModifiedResponse string `json:"modified_response"`
}

type PluginAfterExecutor interface {
	ExecuteAfter(context.Context, PluginAfterExecuteRequest) (*PluginAfterExecuteResponse, error)
}

func (h *Handler) handlePluginExecuteBefore(ctx context.Context, m Message) (*PluginBeforeExecuteResponse, error) {
	var result *PluginBeforeExecuteResponse
	for _, p := range h.cfg.Plugins.Items {
		end := false
		logger := h.logger.WithField("plugin", p.Plugin.PluginName()).WithField("step", "BEFORE")
		err := func() error {
			logger.Debug("Executing plugin")
			defer func() {
				logger.Debug("Finished executing plugin")
			}()

			pbe, ok := p.Plugin.(PluginBeforeExecutor)
			if !ok {
				return nil
			}
			r, err := pbe.ExecuteBefore(ctx, PluginBeforeExecuteRequest(m))
			if err != nil {
				return err
			}

			if result == nil {
				result = r
			} else {
				if r.ModifiedRequest != "" {
					result.ModifiedRequest = r.ModifiedRequest
				}
				result.CustomResponse = append(result.CustomResponse, r.CustomResponse...)
				result.TerminateRequest = r.TerminateRequest
			}

			if !p.AllowedToTerminatePlugins {
				result.TerminatePlugins = false
			}
			if !p.AllowedToTerminateRequest {
				result.TerminateRequest = false
			}

			if result.TerminatePlugins {
				end = true
			}
			return nil
		}()
		if err != nil && !p.IgnoreIfError {
			return nil, err
		}

		if end {
			return result, nil
		}
	}

	return result, nil
}

func (h *Handler) handlePluginExecuteAfter(ctx context.Context, turn *botastic.ConvTurn) (*PluginAfterExecuteResponse, error) {
	var result *PluginAfterExecuteResponse
	for _, p := range h.cfg.Plugins.Items {
		end := false
		logger := h.logger.WithField("plugin", p.Plugin.PluginName()).WithField("step", "AFTER")
		err := func() error {
			logger.Debug("Executing plugin")
			defer func() {
				logger.Debug("Finished executing plugin")
			}()

			pbe, ok := p.Plugin.(PluginAfterExecutor)
			if !ok {
				return nil
			}
			r, err := pbe.ExecuteAfter(ctx, PluginAfterExecuteRequest(*turn))
			if err != nil {
				return err
			}

			if result == nil {
				result = r
			} else {
				if r.ModifiedResponse != "" {
					result.ModifiedResponse = r.ModifiedResponse
				}
			}

			if !p.AllowedToTerminatePlugins {
				result.TerminatePlugins = false
			}

			if result.TerminatePlugins {
				end = true
			}
			return nil
		}()
		if err != nil && !p.IgnoreIfError {
			return nil, err
		}

		if end {
			return result, nil
		}
	}

	return result, nil
}
