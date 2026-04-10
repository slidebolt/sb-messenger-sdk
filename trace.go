package messenger

const (
	HeaderTraceID       = "Sb-Trace-Id"
	HeaderParentTraceID = "Sb-Parent-Trace-Id"
	HeaderOriginService = "Sb-Origin-Service"
	HeaderOriginEntity  = "Sb-Origin-Entity"
	HeaderOriginAction  = "Sb-Origin-Action"
)

func CopyHeaders(headers Headers) Headers {
	if len(headers) == 0 {
		return nil
	}
	out := Headers{}
	for k, v := range headers {
		if k == "" || v == "" {
			continue
		}
		out[k] = v
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func TraceID(headers Headers) string {
	if len(headers) == 0 {
		return ""
	}
	return headers[HeaderTraceID]
}

func WithTraceID(headers Headers, traceID string) Headers {
	out := CopyHeaders(headers)
	if traceID == "" {
		return out
	}
	if out == nil {
		out = Headers{}
	}
	out[HeaderTraceID] = traceID
	return out
}

func WithOrigin(headers Headers, service, entity, action string) Headers {
	out := CopyHeaders(headers)
	if out == nil {
		out = Headers{}
	}
	if service != "" {
		out[HeaderOriginService] = service
	}
	if entity != "" {
		out[HeaderOriginEntity] = entity
	}
	if action != "" {
		out[HeaderOriginAction] = action
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
