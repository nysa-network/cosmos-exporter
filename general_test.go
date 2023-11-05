package main

import (
	"context"
	"fmt"
	"strings"
	"testing"

	govv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	govv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

func queryGov(conn *grpc.ClientConn) error {
	// Handle v1beta1
	govv1beta1Client := govv1beta1.NewQueryClient(conn)
	{
		proposals, err := govv1beta1Client.Proposals(context.Background(), &govv1beta1.QueryProposalsRequest{
			ProposalStatus: govv1beta1.StatusVotingPeriod,
		})
		if err != nil {
			if !strings.Contains(err.Error(), `invalid type: can't convert a gov/v1 Proposal to gov/v1beta1 Proposal`) {
				return err
			}
		} else if err == nil {
			pp := proposals.GetProposals()
			fmt.Printf("proposals: %T\n", proposals)
			fmt.Printf("proposals: %v\n", len(pp))
			return nil
		}
	}

	govv1Client := govv1.NewQueryClient(conn)
	{
		proposals, err := govv1Client.Proposals(context.Background(), &govv1.QueryProposalsRequest{
			ProposalStatus: govv1.StatusVotingPeriod,
		})
		if err != nil {
			return err
		}
		pp := proposals.GetProposals()
		fmt.Printf("proposals: %T\n", proposals)
		fmt.Printf("proposals: %v\n", len(pp))
	}

	return nil
}

func TestGetActiveProposalsJUNO(t *testing.T) {
	conn, err := grpc.Dial(
		"juno-grpc.nysa.network:9090",
		grpc.WithInsecure(),
	)
	require.NoError(t, err)

	defer conn.Close()

	err = queryGov(conn)
	require.NoError(t, err)
}

func TestGetActiveProposalsArchway(t *testing.T) {
	conn, err := grpc.Dial(
		"archway-grpc.nysa.network:9090",
		grpc.WithInsecure(),
	)
	require.NoError(t, err)

	defer conn.Close()

	err = queryGov(conn)
	require.NoError(t, err)
}
