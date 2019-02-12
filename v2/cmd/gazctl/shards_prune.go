package main

import (
	"context"

	"github.com/LiveRamp/gazette/v2/pkg/client"
	"github.com/LiveRamp/gazette/v2/pkg/consumer"
	"github.com/LiveRamp/gazette/v2/pkg/fragment"
	mbp "github.com/LiveRamp/gazette/v2/pkg/mainboilerplate"
	pb "github.com/LiveRamp/gazette/v2/pkg/protocol"
	"github.com/LiveRamp/gazette/v2/pkg/recoverylog"
	log "github.com/sirupsen/logrus"
)

type cmdShardsPrune struct {
	Selector string `long:"selector" short:"l" required:"true" description:"Label Selector query to filter on"`
	DryRun   bool   `long:"dry-run" description:"Perform a dry-run of the apply"`
}

func (cmd *cmdShardsPrune) Execute([]string) error {
	startup()
	var ctx = context.Background()

	var m = metrics{}
	for _, shard := range listShards(cmd.Selector).Shards {
		m.totalShards++
		var lastHints = fetchLastHints(ctx, shard.Spec.Id)
		var _, segments, err = lastHints.LiveLogSegments()
		if err != nil {
			mbp.Must(err, "unable to fetch hint segments")
		}
		for _, sf := range fetchFragments(ctx, lastHints.Log) {
			m.totalFragments++
			m.bytesTotal += sf.ContentLength()

			if len(segments.Intersect(sf.Begin, sf.End)) == 0 {
				log.WithFields(log.Fields{
					"log":  sf.Journal,
					"name": sf.ContentName(),
					"size": sf.ContentLength(),
					"mod":  sf.ModTime,
				}).Warn("pruning fragment")

				m.nPruned++
				m.bytesPruned += sf.ContentLength()

				if !cmd.DryRun {
					fragment.Remove(ctx, sf.BackingStore, sf.Fragment)
				}
			}
		}
		logMetrics(m, shard.Spec.Id.String(), "finished pruning log for shard")
	}
	logMetrics(m, "", "finished pruning log for all shards")
	return nil
}

func fetchLastHints(ctx context.Context, id consumer.ShardID) *recoverylog.FSMHints {
	var err error
	var req = &consumer.GetHintsRequest{
		Shard: id,
	}

	resp, err := consumer.FetchHints(ctx, consumer.NewShardClient(shardsCfg.Consumer.Dial(ctx)), req)
	mbp.Must(err, "failed to list shards")

	for i := len(resp.BackupHints); i != 0; i-- {
		if resp.BackupHints[i] != nil {
			return resp.BackupHints[i]
		}
	}

	mbp.Must(err, "no hints have been written for this shard")
	return nil
}

func fetchFragments(ctx context.Context, journal pb.Journal) []pb.FragmentsResponse_SignedFragment {
	var err error
	var req = pb.FragmentsRequest{
		Journal: journal,
	}
	var brokerClient = journalsCfg.Broker.RoutedJournalClient(ctx)

	resp, err := client.ListAllFragments(ctx, brokerClient, req)
	mbp.Must(err, "failed to list shards")

	return resp.Fragments
}

type metrics struct {
	totalShards    int64
	totalFragments int64
	nPruned        int64
	bytesTotal     int64
	bytesPruned    int64
}

func logMetrics(m metrics, shard, message string) {
	var fields = log.Fields{
		"totalFragments": m.totalFragments,
		"nPruned":        m.nPruned,
		"nLive":          m.totalFragments - m.nPruned,
		"bytesTotal":     m.bytesTotal,
		"bytesPruned":    m.bytesPruned,
		"bytesLive":      m.bytesTotal - m.bytesPruned,
	}
	if shard != "" {
		fields["shard"] = shard
	}
	log.WithFields(fields).Info(message)
}
