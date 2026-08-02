package http

// import (
// 	"context"
// 	"errors"

// 	api "github.com/whicu/hsa/api/http"
// 	"github.com/whicu/hsa/pkg/codec"
// 	"github.com/whicu/hsa/pkg/endpoint"
// )

// type Server struct {

// 	// API
// 	v1InboundMessagePost endpoint.HandlerFunc[*api.InboundMessage, api.V1InboundMessagePostRes]
// }

// var _ api.Handler = (*Server)(nil)

// func New(handler endpoint.HandlerFunc[*api.InboundMessage, api.V1InboundMessagePostRes]) (*Server, error) {
// 	s := &Server{
// 		v1InboundMessagePost: handler,
// 	}

// 	return s, nil
// }

// // NewError implements [api.Handler].
// func (s *Server) NewError(ctx context.Context, err error) *api.ErrorResponseStatusCode {
// 	// Анализируйте тип ошибки
// 	if decodeErr, ok := errors.AsType[*codec.DecodeError](err); ok {
// 		// Ошибка декодирования - 400 Bad Request
// 		return &api.ErrorResponseStatusCode{
// 			StatusCode: 400,
// 			Response: api.ErrorResponse{
// 				Error: api.ErrorResponseError{
// 					Code:    api.ErrorResponseErrorCodeINVALIDPAYLOAD,
// 					Message: decodeErr.Error(),
// 				},
// 			},
// 		}
// 	}

// 	// Дефолтный случай - 500
// 	return &api.ErrorResponseStatusCode{
// 		StatusCode: 500,
// 		Response: api.ErrorResponse{
// 			Error: api.ErrorResponseError{
// 				Code:    api.ErrorResponseErrorCodeINTERNALERROR,
// 				Message: err.Error(),
// 			},
// 		},
// 	}
// }

// // V1InboundMessagePost implements [api.Handler].
// func (s *Server) V1InboundMessagePost(ctx context.Context, req *api.InboundMessage, params api.V1InboundMessagePostParams) (api.V1InboundMessagePostRes, error) {
// 	return s.v1InboundMessagePost(ctx, req)
// }
