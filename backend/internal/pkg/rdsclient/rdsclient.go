// Package rdsclient is a thin wrapper around aws-sdk-go-v2's RDS client,
// implementing domain.DBInstanceClient for the idle auto-stop/wake-on-hit
// feature (spec 010, db-idle-autostop). It authenticates via the EC2
// instance's IAM role (SDK default credential chain, no static keys —
// research.md Decision 3), mirroring internal/pkg/attachmentstorage's shape.
// Region is NOT auto-resolved from EC2 instance metadata the way credentials
// are — an AWS_REGION env var must be set in the deployment environment, or
// New's LoadDefaultConfig call fails with "missing region" (research.md
// Decision 7, corrected after a real deployment hit this exact crash).
package rdsclient

import (
	"context"
	"fmt"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/rds/types"

	"github.com/sharique/mansooba/internal/domain"
)

// Client wraps the AWS RDS client for one specific database instance.
type Client struct {
	rds        *rds.Client
	instanceID string
}

// New returns a Client for the given RDS instance identifier, using the
// AWS SDK's default credential chain (the EC2 instance role in production;
// this package is never exercised in local dev, which uses SQLite).
func New(ctx context.Context, instanceID string) (*Client, error) {
	cfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}
	return &Client{
		rds:        rds.NewFromConfig(cfg),
		instanceID: instanceID,
	}, nil
}

// StartDBInstance requests the instance start. This is a fast, fire-and-ack
// call — it does not wait for the instance to become available.
func (c *Client) StartDBInstance(ctx context.Context) error {
	_, err := c.rds.StartDBInstance(ctx, &rds.StartDBInstanceInput{
		DBInstanceIdentifier: &c.instanceID,
	})
	return err
}

// StopDBInstance requests the instance stop. Like StartDBInstance, this only
// requests the transition; it does not wait for it to complete.
func (c *Client) StopDBInstance(ctx context.Context) error {
	_, err := c.rds.StopDBInstance(ctx, &rds.StopDBInstanceInput{
		DBInstanceIdentifier: &c.instanceID,
	})
	return err
}

// DescribeState returns the instance's current lifecycle state, mapped from
// AWS RDS's DBInstanceStatus string onto domain.DBInstanceState. Any status
// not explicitly recognized here (RDS has many transient/maintenance
// statuses beyond the four this feature cares about) maps to
// DBInstanceRunning, since those all represent the instance being up and
// usable in some form, not stopped or mid stop/start transition.
func (c *Client) DescribeState(ctx context.Context) (domain.DBInstanceState, error) {
	out, err := c.rds.DescribeDBInstances(ctx, &rds.DescribeDBInstancesInput{
		DBInstanceIdentifier: &c.instanceID,
	})
	if err != nil {
		return domain.DBInstanceRunning, err
	}
	if len(out.DBInstances) == 0 {
		return domain.DBInstanceRunning, fmt.Errorf("rdsclient: no instance found for identifier %q", c.instanceID)
	}
	return mapStatus(out.DBInstances[0]), nil
}

func mapStatus(inst types.DBInstance) domain.DBInstanceState {
	if inst.DBInstanceStatus == nil {
		return domain.DBInstanceRunning
	}
	switch *inst.DBInstanceStatus {
	case "stopped":
		return domain.DBInstanceStopped
	case "stopping":
		return domain.DBInstanceStopping
	case "starting":
		return domain.DBInstanceStarting
	default:
		return domain.DBInstanceRunning
	}
}
