package encoders

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"

	"code.cloudfoundry.org/lager"
	"github.com/quizlet/lame"

	"github.com/snickers/snickers/db"
	"github.com/snickers/snickers/types"
)

func LAMEEncode(logger lager.Logger, dbInstance db.Storage, jobID string) error {
	log := logger.Session("lame-encode")
	log.Info("started", lager.Data{"job": jobID})
	defer log.Info("finished")

	job, _ := dbInstance.RetrieveJob(jobID)

	f, err := os.Open(job.LocalSource)
	if err != nil {
		log.Error("input-failed", err)
		return err
	}
	defer f.Close()

	reader := bufio.NewReader(f)

	of, err := os.Create(job.LocalDestination)
	if err != nil {
		log.Error("output-failed", err)
		return err
	}
	defer of.Close()

	job.Status = types.JobEncoding
	job.Progress = "0%"
	dbInstance.UpdateJob(job.ID, job)

	wr := lame.NewWriter(of)

	if job.Preset.Audio.Mode != "" {
		fmt.Printf("Setting Mode to %s", job.Preset.Audio.Mode)
		switch job.Preset.Audio.Mode {
		case "STEREO":
			wr.Encoder.SetMode(lame.STEREO)
		case "JOINT_STEREO":
			wr.Encoder.SetMode(lame.JOINT_STEREO)
		case "MONO":
			wr.Encoder.SetMode(lame.MONO)
		default:
			return errors.New(fmt.Sprintf("%s mode not supported.", job.Preset.Audio.Mode))
		}
	}

	if job.Preset.Audio.Bitrate != "" {
		if job.Preset.RateControl == "vbr" {
			re := regexp.MustCompile("v([0-9])")
			matched := re.FindStringSubmatch(job.Preset.Audio.Bitrate)

			if len(matched) == 2 {
				i, err := strconv.Atoi(matched[1])
				if err == nil {
					log.Info("set-vbr-quality", lager.Data{"quality": i})
					wr.Encoder.SetVBRQuality(i)
					wr.Encoder.SetVBR(lame.VBR_DEFAULT)
				}
			}
		} else {
			i, err := strconv.Atoi(job.Preset.Audio.Bitrate)
			if err == nil {
				wr.Encoder.SetBitrate(i)
			}
		}
	}

	if job.Preset.Audio.Quality != "" {
		i, err := strconv.Atoi(job.Preset.Audio.Quality)
		if err == nil {
			wr.Encoder.SetQuality(i)
		}
	}

	// IMPORTANT!
	wr.Encoder.InitParams()

	reader.WriteTo(wr)

	if job.Progress != "100%" {
		job.Progress = "100%"
		dbInstance.UpdateJob(job.ID, job)
	}

	return nil
}
