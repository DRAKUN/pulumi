// Copyright 2016 Pulumi, Inc. All rights reserved.

package ec2

import (
	"errors"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	pbempty "github.com/golang/protobuf/ptypes/empty"
	"github.com/pulumi/coconut/pkg/resource"
	"github.com/pulumi/coconut/pkg/tokens"
	"github.com/pulumi/coconut/pkg/util/contract"
	"github.com/pulumi/coconut/sdk/go/pkg/cocorpc"
	"golang.org/x/net/context"

	"github.com/pulumi/coconut/lib/aws/provider/awsctx"
)

const Instance = tokens.Type("aws:ec2/instance:Instance")

// NewInstanceProvider creates a provider that handles EC2 instance operations.
func NewInstanceProvider(ctx *awsctx.Context) cocorpc.ResourceProviderServer {
	return &instanceProvider{ctx}
}

type instanceProvider struct {
	ctx *awsctx.Context
}

// Check validates that the given property bag is valid for a resource of the given type.
func (p *instanceProvider) Check(ctx context.Context, req *cocorpc.CheckRequest) (*cocorpc.CheckResponse, error) {
	// Read in the properties, deserialize them, and verify them; return the resulting failures if any.
	contract.Assert(req.GetType() == string(Instance))
	props := resource.UnmarshalProperties(req.GetProperties())
	instance, failures, _ := newInstance(props, true)

	if instance.ImageID != "" {
		// Check that the AMI exists; this catches misspellings, AMI region mismatches, accessibility problems, etc.
		result, err := p.ctx.EC2().DescribeImages(&ec2.DescribeImagesInput{
			ImageIds: []*string{aws.String(instance.ImageID)},
		})
		if err != nil {
			return nil, err
		}
		if len(result.Images) == 0 {
			return nil, fmt.Errorf("missing image: %v", instance.ImageID)
		}
		contract.Assertf(len(result.Images) == 1, "Did not expect multiple instance matches")
		contract.Assertf(result.Images[0].ImageId != nil, "Expected a non-nil matched instance ID")
		contract.Assertf(*result.Images[0].ImageId == instance.ImageID, "Expected instance IDs to match")
	}

	return &cocorpc.CheckResponse{Failures: failures}, nil
}

// Name names a given resource.  Sometimes this will be assigned by a developer, and so the provider
// simply fetches it from the property bag; other times, the provider will assign this based on its own algorithm.
// In any case, resources with the same name must be safe to use interchangeably with one another.
func (p *instanceProvider) Name(ctx context.Context, req *cocorpc.NameRequest) (*cocorpc.NameResponse, error) {
	return nil, nil // use the AWS provider default name
}

// Create allocates a new instance of the provided resource and returns its unique ID afterwards.  (The input ID
// must be blank.)  If this call fails, the resource must not have been created (i.e., it is "transacational").
func (p *instanceProvider) Create(ctx context.Context, req *cocorpc.CreateRequest) (*cocorpc.CreateResponse, error) {
	contract.Assert(req.GetType() == string(Instance))
	props := resource.UnmarshalProperties(req.GetProperties())

	// Read in the properties given by the request, validating as we go; if any fail, reject the request.
	// TODO: validate additional properties (e.g., that AMI exists in this region).
	// TODO: this is a good example of a "benign" (StateOK) error; handle it accordingly.
	inst, _, err := newInstance(props, true)
	if err != nil {
		return nil, err
	}

	// Create the create instances request object.
	var secgrpIDs []*string
	if inst.SecurityGroupIDs != nil {
		secgrpIDs = aws.StringSlice(*inst.SecurityGroupIDs)
	}
	create := &ec2.RunInstancesInput{
		ImageId:          aws.String(inst.ImageID),
		InstanceType:     inst.InstanceType,
		SecurityGroupIds: secgrpIDs,
		KeyName:          inst.KeyName,
		MinCount:         aws.Int64(int64(1)),
		MaxCount:         aws.Int64(int64(1)),
	}

	// Now go ahead and perform the action.
	fmt.Fprintf(os.Stdout, "Creating new EC2 instance resource\n")
	out, err := p.ctx.EC2().RunInstances(create)
	if err != nil {
		return nil, err
	}
	contract.Assert(out != nil)
	contract.Assert(len(out.Instances) == 1)
	contract.Assert(out.Instances[0] != nil)
	contract.Assert(out.Instances[0].InstanceId != nil)

	// Before returning that all is okay, wait for the instance to reach the running state.
	id := out.Instances[0].InstanceId
	fmt.Fprintf(os.Stdout, "EC2 instance '%v' created; now waiting for it to become 'running'\n", *id)
	// TODO: consider a custom wait function so that we can have uniformity across all of our providers.
	// TODO: if this fails, but the creation succeeded, we will have an orphaned resource; report this differently.
	err = p.ctx.EC2().WaitUntilInstanceRunning(
		&ec2.DescribeInstancesInput{InstanceIds: []*string{id}})

	return &cocorpc.CreateResponse{
		Id: *id,
	}, err
}

// Read reads the instance state identified by ID, returning a populated resource object, or an error if not found.
func (p *instanceProvider) Read(ctx context.Context, req *cocorpc.ReadRequest) (*cocorpc.ReadResponse, error) {
	contract.Assert(req.GetType() == string(Instance))
	return nil, errors.New("Not yet implemented")
}

// Update updates an existing resource with new values.  Only those values in the provided property bag are updated
// to new values.  The resource ID is returned and may be different if the resource had to be recreated.
func (p *instanceProvider) Update(ctx context.Context, req *cocorpc.UpdateRequest) (*pbempty.Empty, error) {
	contract.Assert(req.GetType() == string(Instance))
	return nil, errors.New("No known updatable instance properties")
}

// UpdateImpact checks what impacts a hypothetical update will have on the resource's properties.
func (p *instanceProvider) UpdateImpact(
	ctx context.Context, req *cocorpc.UpdateRequest) (*cocorpc.UpdateImpactResponse, error) {
	contract.Assert(req.GetType() == string(Instance))
	// Unmarshal and validate the old and new properties.
	olds := resource.UnmarshalProperties(req.GetOlds())
	if _, _, err := newInstance(olds, true); err != nil {
		return nil, err
	}
	news := resource.UnmarshalProperties(req.GetNews())
	if _, _, err := newInstance(news, true); err != nil {
		return nil, err
	}

	// Now check the diff for updates to any fields (none of them are updateable).
	var replaces []string
	diff := olds.Diff(news)
	if diff.Changed(instanceImageID) {
		replaces = append(replaces, instanceImageID)
	}
	if diff.Changed(instanceType) {
		replaces = append(replaces, instanceType)
	}
	if diff.Changed(instanceKeyName) {
		replaces = append(replaces, instanceKeyName)
	}
	if diff.Changed(instanceSecurityGroups) {
		// TODO: we should permit changes to security groups for non-EC2-classic VMs that are in VPCs.
		replaces = append(replaces, instanceSecurityGroups)
	}
	return &cocorpc.UpdateImpactResponse{Replaces: replaces}, nil
}

// Delete tears down an existing resource with the given ID.  If it fails, the resource is assumed to still exist.
func (p *instanceProvider) Delete(ctx context.Context, req *cocorpc.DeleteRequest) (*pbempty.Empty, error) {
	contract.Assert(req.GetType() == string(Instance))
	id := aws.String(req.GetId())
	delete := &ec2.TerminateInstancesInput{
		InstanceIds: []*string{id},
	}
	fmt.Fprintf(os.Stdout, "Terminating EC2 instance '%v'\n", *id)
	if _, err := p.ctx.EC2().TerminateInstances(delete); err != nil {
		return nil, err
	}
	fmt.Fprintf(os.Stdout, "EC2 instance termination request submitted; waiting for it to terminate\n")
	err := p.ctx.EC2().WaitUntilInstanceTerminated(
		&ec2.DescribeInstancesInput{InstanceIds: []*string{id}})
	return &pbempty.Empty{}, err
}

// instance represents the state associated with an instance.
type instance struct {
	ImageID          string
	InstanceType     *string
	SecurityGroupIDs *[]string
	KeyName          *string
}

const (
	instanceImageID        = "imageId"
	instanceType           = "instanceType"
	instanceSecurityGroups = "securityGroups"
	instanceKeyName        = "keyName"
)

// newInstance creates a new instance bag of state, validating required properties if asked to do so.
func newInstance(m resource.PropertyMap, req bool) (*instance, []*cocorpc.CheckFailure, error) {
	var failures []*cocorpc.CheckFailure

	id, err := m.ReqStringOrErr(instanceImageID)
	if err != nil && (req || !resource.IsReqError(err)) {
		failures = append(failures,
			&cocorpc.CheckFailure{Property: instanceImageID, Reason: err.Error()})
	}

	t, err := m.OptStringOrErr(instanceType)
	if err != nil {
		failures = append(failures,
			&cocorpc.CheckFailure{Property: instanceType, Reason: err.Error()})
	}

	securityGroupIDs, err := m.OptStringArrayOrErr(instanceSecurityGroups)
	if err != nil {
		failures = append(failures,
			&cocorpc.CheckFailure{Property: instanceSecurityGroups, Reason: err.Error()})
	}

	keyName, err := m.OptStringOrErr(instanceKeyName)
	if err != nil {
		failures = append(failures,
			&cocorpc.CheckFailure{Property: instanceKeyName, Reason: err.Error()})
	}

	return &instance{
		ImageID:          id,
		InstanceType:     t,
		SecurityGroupIDs: securityGroupIDs,
		KeyName:          keyName,
	}, failures, awsctx.MaybeCheckError(failures)
}
