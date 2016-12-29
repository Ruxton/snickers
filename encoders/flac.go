package encoders

import (
	"fmt"
	"os"

	"code.cloudfoundry.org/lager"
	"github.com/cocoonlife/goflac"
	"github.com/youpy/go-wav"

	"github.com/snickers/snickers/db"
	"github.com/snickers/snickers/types"
)

func FLACEncode(logger lager.Logger, dbInstance db.Storage, jobID string) error {
	log := logger.Session("flac-encode")
	log.Info("started", lager.Data{"job": jobID})
	defer log.Info("finished")

	job, _ := dbInstance.RetrieveJob(jobID)

	f, err := os.Open(job.LocalSource)
	if err != nil {
		log.Error("input-failed", err)
		return err
	}
	defer f.Close()

	job.Status = types.JobEncoding
	job.Progress = "0%"
	dbInstance.UpdateJob(job.ID, job)

	enc, err := libflac.NewEncoder(job.LocalDestination, 2, 16, 44100)
	fmt.Println("SETUP ENCODERWRITER")

	reader := wav.NewReader(f)

	var out []int32
	var samples []wav.Sample
	var count int32 = 0
	for {
		samples, err = reader.ReadSamples()
		if err != nil {
			enc.WriteFrame(libflac.Frame{2, 16, 44100, out})
			fmt.Printf("Read error - %s\n", err)
			break
		}
		for _, sample := range samples {
			out = append(out, int32(reader.IntValue(sample, 0)))
			out = append(out, int32(reader.IntValue(sample, 1)))
		}
		if count == 200 {
			enc.WriteFrame(libflac.Frame{2, 16, 44100, out})
			out = nil
			count = 0
		} else {
			count = count + 1
		}
	}

	enc.Close()
	fmt.Println("CLOSING")

	if job.Progress != "100%" {
		job.Progress = "100%"
		dbInstance.UpdateJob(job.ID, job)
	}

	return nil
}
