package policy

import "github.com/rs/zerolog"

type InvalidCollectorJobError struct {
	InvalidSpecsByReprtingJob     map[string][]string
	InvalidNotifiersByReprtingJob map[string][]string
}

func newInvalidCollectorJobError() *InvalidCollectorJobError {
	return &InvalidCollectorJobError{
		InvalidSpecsByReprtingJob:     make(map[string][]string),
		InvalidNotifiersByReprtingJob: make(map[string][]string),
	}
}

func (e *InvalidCollectorJobError) addInvalidSpec(jobUUID string, specRef string) {
	e.InvalidSpecsByReprtingJob[jobUUID] = append(e.InvalidSpecsByReprtingJob[jobUUID], specRef)
}

func (e *InvalidCollectorJobError) addInvalidNotifier(jobUUID string, jobRef string) {
	e.InvalidNotifiersByReprtingJob[jobUUID] = append(e.InvalidNotifiersByReprtingJob[jobUUID], jobRef)
}

func (e *InvalidCollectorJobError) Error() string {
	return "invalid collector job"
}

func (e *InvalidCollectorJobError) MarshalZerologObject(ev *zerolog.Event) {
	specsByJob := zerolog.Dict()
	for rjUUID, specRefs := range e.InvalidSpecsByReprtingJob {
		specsByJob.Strs(rjUUID, specRefs)
	}
	ev.Dict("specs-by-job", specsByJob)

	notifiersByJob := zerolog.Dict()
	for rjUUID, notifierRef := range e.InvalidNotifiersByReprtingJob {
		notifiersByJob.Strs(rjUUID, notifierRef)
	}
	ev.Dict("notifiers-by-job", notifiersByJob)
}

// Validate that this collector job is wired correctly and adheres to internal consistency
func (c *CollectorJob) Validate() error {
	invalid := false
	invalidCollectorJobError := newInvalidCollectorJobError()

	for _, job := range c.ReportingJobs {
		for ref := range job.Spec {
			if _, ok := c.ReportingJobs[ref]; !ok {
				invalid = true
				invalidCollectorJobError.addInvalidSpec(job.Uuid, ref)
			}
		}

		for ref := range job.Notify {
			ref := job.Notify[ref]
			if _, ok := c.ReportingJobs[ref]; !ok {
				invalid = true
				invalidCollectorJobError.addInvalidNotifier(job.Uuid, ref)
			}
		}
	}

	if !invalid {
		return nil
	}

	return invalidCollectorJobError
}
