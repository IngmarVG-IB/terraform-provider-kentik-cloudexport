package kentikapitest

import (
	"context"
	"strconv"
	"sync"

	cloudexport "github.com/kentik/api-schema-public/gen/go/kentik/cloud_export/v202506"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// CloudExportServer is an in-memory CloudExportAdminService. It behaves like the
// real API where it matters for the provider: it assigns IDs, timestamps and a
// status on create, replaces the whole object on update, rejects duplicate names
// and returns NotFound for unknown IDs. All methods are safe for concurrent use.
type CloudExportServer struct {
	cloudexport.UnimplementedCloudExportAdminServiceServer

	mu      sync.Mutex
	exports []*cloudexport.CloudExport
	nextID  int

	// FailNext, if set, makes the next N RPCs fail with the given code.
	failCode  codes.Code
	failCount int
}

// NewCloudExportServer returns a fake seeded with copies of exports.
func NewCloudExportServer(exports ...*cloudexport.CloudExport) *CloudExportServer {
	s := &CloudExportServer{}
	for _, e := range exports {
		s.exports = append(s.exports, proto.CloneOf(e))
		if id, err := strconv.Atoi(e.GetId()); err == nil && id > s.nextID {
			s.nextID = id
		}
	}
	return s
}

// Register registers the fake on srv.
func (s *CloudExportServer) Register(srv *Server) *CloudExportServer {
	cloudexport.RegisterCloudExportAdminServiceServer(srv.Registrar(), s)
	return s
}

// FailNext makes the next n RPCs fail with code.
func (s *CloudExportServer) FailNext(n int, code codes.Code) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.failCount, s.failCode = n, code
}

// Get returns a copy of the stored export with id, or nil.
func (s *CloudExportServer) Get(id string) *cloudexport.CloudExport {
	s.mu.Lock()
	defer s.mu.Unlock()
	if i := s.find(id); i >= 0 {
		return proto.CloneOf(s.exports[i])
	}
	return nil
}

// Len returns the number of stored exports.
func (s *CloudExportServer) Len() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.exports)
}

func (s *CloudExportServer) injectedFailure() error {
	if s.failCount > 0 {
		s.failCount--
		return status.Error(s.failCode, "injected failure")
	}
	return nil
}

func (s *CloudExportServer) find(id string) int {
	for i, e := range s.exports {
		if e.GetId() == id {
			return i
		}
	}
	return -1
}

// ListCloudExports implements CloudExportAdminServiceServer.
func (s *CloudExportServer) ListCloudExports(
	context.Context, *cloudexport.ListCloudExportsRequest,
) (*cloudexport.ListCloudExportsResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.injectedFailure(); err != nil {
		return nil, err
	}
	resp := &cloudexport.ListCloudExportsResponse{}
	for _, e := range s.exports {
		resp.Exports = append(resp.Exports, proto.CloneOf(e))
	}
	return resp, nil
}

// GetCloudExport implements CloudExportAdminServiceServer.
func (s *CloudExportServer) GetCloudExport(
	_ context.Context, req *cloudexport.GetCloudExportRequest,
) (*cloudexport.GetCloudExportResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.injectedFailure(); err != nil {
		return nil, err
	}
	if i := s.find(req.GetId()); i >= 0 {
		return &cloudexport.GetCloudExportResponse{Export: proto.CloneOf(s.exports[i])}, nil
	}
	return nil, status.Errorf(codes.NotFound, "cloud export %q not found", req.GetId())
}

// CreateCloudExport implements CloudExportAdminServiceServer.
func (s *CloudExportServer) CreateCloudExport(
	_ context.Context, req *cloudexport.CreateCloudExportRequest,
) (*cloudexport.CreateCloudExportResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.injectedFailure(); err != nil {
		return nil, err
	}
	e := proto.CloneOf(req.GetExport())
	if err := validate(e); err != nil {
		return nil, err
	}
	for _, other := range s.exports {
		if other.GetName() == e.GetName() {
			return nil, status.Errorf(codes.AlreadyExists, "cloud export %q already exists", e.GetName())
		}
	}
	s.nextID++
	e.Id = strconv.Itoa(s.nextID)
	now := timestamppb.Now()
	e.Cdate, e.Edate = now, now
	e.CurrentStatus = &cloudexport.CloudExportStatus{Status: "OK", FlowFound: true, ApiAccess: true, StorageAccountAccess: true}
	applyServerDefaults(e)
	s.exports = append(s.exports, e)
	return &cloudexport.CreateCloudExportResponse{Export: proto.CloneOf(e)}, nil
}

// UpdateCloudExport implements CloudExportAdminServiceServer.
func (s *CloudExportServer) UpdateCloudExport(
	_ context.Context, req *cloudexport.UpdateCloudExportRequest,
) (*cloudexport.UpdateCloudExportResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.injectedFailure(); err != nil {
		return nil, err
	}
	e := proto.CloneOf(req.GetExport())
	i := s.find(e.GetId())
	if i < 0 {
		return nil, status.Errorf(codes.NotFound, "cloud export %q not found", e.GetId())
	}
	if err := validate(e); err != nil {
		return nil, err
	}
	old := s.exports[i]
	e.Cdate, e.Edate, e.CurrentStatus = old.GetCdate(), timestamppb.Now(), old.GetCurrentStatus()
	applyServerDefaults(e)
	s.exports[i] = e
	return &cloudexport.UpdateCloudExportResponse{Export: proto.CloneOf(e)}, nil
}

// DeleteCloudExport implements CloudExportAdminServiceServer.
func (s *CloudExportServer) DeleteCloudExport(
	_ context.Context, req *cloudexport.DeleteCloudExportRequest,
) (*cloudexport.DeleteCloudExportResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.injectedFailure(); err != nil {
		return nil, err
	}
	i := s.find(req.GetId())
	if i < 0 {
		return nil, status.Errorf(codes.NotFound, "cloud export %q not found", req.GetId())
	}
	s.exports = append(s.exports[:i], s.exports[i+1:]...)
	return &cloudexport.DeleteCloudExportResponse{}, nil
}

func validate(e *cloudexport.CloudExport) error {
	switch {
	case e.GetName() == "":
		return status.Error(codes.InvalidArgument, "name is required")
	case e.GetPlanId() == "":
		return status.Error(codes.InvalidArgument, "plan_id is required")
	case e.GetCloudProvider() == cloudexport.CloudProvider_CLOUD_PROVIDER_UNSPECIFIED:
		return status.Error(codes.InvalidArgument, "cloud_provider is required")
	}
	return nil
}

// applyServerDefaults mimics the documented API defaults for proto3 optional
// fields that the client left unset, so tests see server-computed values.
func applyServerDefaults(e *cloudexport.CloudExport) {
	if aws := e.GetAws(); aws != nil {
		if aws.MetadataOnly == nil {
			aws.MetadataOnly = proto.Bool(false)
		}
		if aws.CollectFlowLogs == nil {
			aws.CollectFlowLogs = proto.Bool(true)
		}
		if aws.CollectFirewallLogs == nil {
			aws.CollectFirewallLogs = proto.Bool(false)
		}
		if aws.CollectMetrics == nil {
			aws.CollectMetrics = proto.Bool(false)
		}
	}
	if e.GetSampling() == nil {
		e.Sampling = &cloudexport.CloudExportSamplingProperties{
			SamplingType: cloudexport.CloudExportSamplingType_CLOUD_EXPORT_SAMPLING_TYPE_UNSAMPLED,
		}
	}
}
