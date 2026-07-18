package spicedbproxy

import (
	"context"
	"crypto/subtle"
	"strings"

	v1 "github.com/authzed/authzed-go/proto/authzed/api/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type Proxy struct {
	v1.UnimplementedPermissionsServiceServer
	v1.UnimplementedSchemaServiceServer
	permissions v1.PermissionsServiceClient
	schema      v1.SchemaServiceClient
}

func New(permissions v1.PermissionsServiceClient, schema v1.SchemaServiceClient) *Proxy {
	return &Proxy{permissions: permissions, schema: schema}
}

func (p *Proxy) WriteRelationships(ctx context.Context, request *v1.WriteRelationshipsRequest) (*v1.WriteRelationshipsResponse, error) {
	return p.permissions.WriteRelationships(ctx, request)
}
func (p *Proxy) CheckPermission(ctx context.Context, request *v1.CheckPermissionRequest) (*v1.CheckPermissionResponse, error) {
	return p.permissions.CheckPermission(ctx, request)
}
func (p *Proxy) ReadSchema(ctx context.Context, request *v1.ReadSchemaRequest) (*v1.ReadSchemaResponse, error) {
	return p.schema.ReadSchema(ctx, request)
}

func (p *Proxy) Register(server *grpc.Server) {
	v1.RegisterPermissionsServiceServer(server, p)
	v1.RegisterSchemaServiceServer(server, p)
}

func Authenticate(runtimeToken string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, request any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		values := metadata.ValueFromIncomingContext(ctx, "authorization")
		if len(values) != 1 || !constantTimeToken(strings.TrimPrefix(values[0], "Bearer "), runtimeToken) {
			return nil, status.Error(codes.Unauthenticated, "runtime credential rejected")
		}
		switch info.FullMethod {
		case "/authzed.api.v1.PermissionsService/WriteRelationships", "/authzed.api.v1.PermissionsService/CheckPermission", "/authzed.api.v1.SchemaService/ReadSchema":
			return handler(ctx, request)
		default:
			return nil, status.Error(codes.PermissionDenied, "operation is outside the runtime capability")
		}
	}
}

func constantTimeToken(actual, expected string) bool {
	return len(actual) == len(expected) && subtle.ConstantTimeCompare([]byte(actual), []byte(expected)) == 1
}
