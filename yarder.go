package main

import (
	"bytes"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/caarlos0/env"
)

// Version is the current version of the software
const VERSION = "0.0.1"

type envData struct {
	Duration   string `env:"YARDER_DURATION" envDefault:"1m"`
	S3Path     string `env:"YARDER_S3_PATH"`
	S3Bucket   string `env:"YARDER_S3_BUCKET"`
	AWSRegion  string `env:"YARDER_AWS_REGION" envDefault:"us-east-1"`
	OutputFile string `env:"YARDER_OUTPUT_FILE"`
	LogFile    string `env:"YARDER_LOG_FILE"`
}

type config struct {
	outputFile string
	s3Path     string
	s3Bucket   string
	awsRegion  string
	duration   time.Duration
	fileName   string
	logFile    string
}

func parseConfig() (*config, error) {
	e := &envData{}
	env.Parse(e)
	// Parse the Length
	duration, err := time.ParseDuration(e.Duration)
	if err != nil {
		return nil, err
	}

	cfg := &config{
		duration:   duration,
		awsRegion:  e.AWSRegion,
		s3Path:     e.S3Path,
		s3Bucket:   e.S3Bucket,
		outputFile: e.OutputFile,
		logFile:    e.LogFile,
	}

	return cfg, nil
}

func tailLog(logFile string, tailFile string) (*exec.Cmd, error) {
	cmd := exec.Command("tail", "-f", logFile)
	f, err := os.Create(tailFile)
	if err != nil {
		return nil, err
	}

	cmd.Stdout = f
	err = cmd.Start()
	if err != nil {
		return nil, err
	}

	return cmd, nil
}

func gzipFile(tailFile string) (string, error) {
	p, file := path.Split(tailFile)
	gzFile := file + ".tar.gz"
	cmd := exec.Command("tar", "-czf", gzFile, file)
	cmd.Dir = p
	err := cmd.Run()
	if err != nil {
		return "", err
	}
	output, err := cmd.Output()
	log.Println(string(output))
	return path.Join(p, gzFile), nil
}

func uploadToS3(gzipFile string, bucket string, s3Path string) error {
	s3Client := s3.New(&aws.Config{Region: aws.String("us-east-1")}) // Rely entirely on environmental config
	file, err := os.Open(gzipFile)
	if err != nil {
		return err
	}

	defer file.Close()
	fileInfo, err := file.Stat()
	if err != nil {
		return err
	}

	buffer := make([]byte, fileInfo.Size())
	_, err = file.Read(buffer)
	if err != nil {
		return err
	}
	reader := bytes.NewReader(buffer)
	fileType := http.DetectContentType(buffer)
	p := path.Join(s3Path, fileInfo.Name())

	params := &s3.PutObjectInput{
		Bucket:        aws.String(bucket),
		Key:           aws.String(p),
		Body:          reader,
		ContentLength: aws.Int64(fileInfo.Size()),
		ContentType:   aws.String(fileType),
		Metadata: map[string]*string{
			"Key": aws.String("MetadataValue"),
		},
	}

	_, err = s3Client.PutObject(params)

	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			// Generic error, info if available
			log.Println(awsErr.Code(), awsErr.Message(), awsErr.OrigErr())
			if reqErr, ok := err.(awserr.RequestFailure); ok {
				// A service error occurred
				log.Println(reqErr.Code(), reqErr.Message(), reqErr.StatusCode(), reqErr.RequestID())
			} else {
				// This case should never be hit, the SDK should always return an
				// error which satisfies the awserr.Error interface.
				log.Println(err.Error())
			}
		}
	}
	return err
}

func main() {
	log.Println("Yarder " + VERSION)
	// Grab required configuration values from env
	cfg, err := parseConfig()
	log.Print(cfg)
	if err != nil {
		log.Println("There was an error configuring Yarder. Please evaluate the error and try again")
		log.Fatal(err)
	}
	// Kick off tail task, get the cmd
	cmd, err := tailLog(cfg.logFile, cfg.outputFile)
	if err != nil {
		log.Println("There was an error during the tail process. Please evaluate the error and try again")
		log.Fatal(err)
	}
	// Wait for the configured timeout
	timer := time.NewTimer(cfg.duration)
	<-timer.C
	// Kill tail task
	err = cmd.Process.Kill()
	if err != nil {
		log.Println("There was an error finishing the tail process. Please evaluate the error and try again")
		log.Fatal(err)
	}
	// Gzip the log data
	gzFile, err := gzipFile(cfg.outputFile)
	if err != nil {
		log.Println("There was an error zipping the tailled log. Please evaluate the error and try again")
		log.Fatal(err)
	}

	// Ship it to s3 in a bucket/path for the user
	err = uploadToS3(gzFile, cfg.s3Bucket, cfg.s3Path)
	// Exit
}
