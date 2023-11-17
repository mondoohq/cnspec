// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package scan

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"
	"go.mondoo.com/cnspec/v9/policy/scan/pdque"
)

type diskQueueConfig struct {
	dir         string
	filename    string
	maxSize int
	sync        bool
}

var defaultDqueConfig = diskQueueConfig{
	dir:         "/tmp/cnspec-queue", // TODO: consider configurable path
	filename:    "disk-queue",
	maxSize: 500,
	sync:        false,
}

// queueMsg is the being stored in disk queue
type queueMsg struct {
	Ts      time.Time
	Payload queuePayload
}

// queuePayload hold the data for the job
type queuePayload struct {
	ScanJob []byte // proto encoded scan job
}

type diskQueueClient struct {
	queue   *pdque.Queue
	once    sync.Once
	wg      sync.WaitGroup
	entries chan Job
	handler func(job *Job)
}

func diskQueueEntryBuilder() interface{} {
	return &queueMsg{}
}

// newDqueClient creates a new dque client which stores scan job events on disk
// It is a First In First Out (FIFO) queue
//
// To push a scan job onto the queue, just place a new *ScanJob onto go channel
// dqueClient.Channel() <- *job
func newDqueClient(config diskQueueConfig, handler func(job *Job)) (*diskQueueClient, error) {
	var err error

	q := &diskQueueClient{
		handler: handler,
	}

	err = os.MkdirAll(config.dir, 0o700)
	if err != nil {
		return nil, fmt.Errorf("cannot create queue directory: %s", err)
	}

	q.queue, err = pdque.NewOrOpen(config.filename, config.dir, config.maxSize, diskQueueEntryBuilder)
	if err != nil {
		return nil, err
	}

	q.entries = make(chan Job)

	q.wg.Add(2)
	go q.pusher()
	go q.popper()
	return q, nil
}

// Stop closes the client
func (c *diskQueueClient) Stop() {
	c.once.Do(func() {
		close(c.entries)
		c.queue.Close()
		c.wg.Wait()
	})
}

func (c *diskQueueClient) Channel() chan<- Job {
	return c.entries
}

// pusher iterates over all new ScanJob events and places them into the disk queue
func (c *diskQueueClient) pusher() {
	defer c.wg.Done()
	for sj := range c.entries {

		// we marshal the scan job since dque uses gob marshaling which would fail otherwise
		data, err := proto.Marshal(&sj)
		if err != nil {
			log.Warn().Err(err).Msg("cannot marshal scan job")
			continue
		}

		err = c.queue.Enqueue(&queueMsg{time.Now(), queuePayload{
			ScanJob: data,
		}})
		if err != nil {
			log.Warn().Err(err).Msg("cannot push scan job on disk queue")
		}
	}
}

// popper reads from the disk queue and runs the handle function per event
func (c *diskQueueClient) popper() {
	defer c.wg.Done()
	for {
		// pop next item from queue
		entry, err := c.queue.DequeueBlock()
		if err != nil {
			switch err {
			case pdque.ErrQueueClosed:
				return
			default:
				log.Error().Err(err).Msg("could not pop job from disk queue")
				continue
			}
		}

		// convert the event into our own event
		record, ok := entry.(*queueMsg)
		if !ok {
			log.Error().Msg("invalid type stored in queue")
			continue
		}

		// unmarshal the protobuf content into scan job and run handler
		var scanJob Job
		err = proto.Unmarshal(record.Payload.ScanJob, &scanJob)
		if err != nil {
			log.Error().Err(err).Msg("could not unmarshal the scan job")
			continue
		}
		c.handler(&scanJob)
	}
}
