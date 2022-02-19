package inputreader

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/mitchellh/go-homedir"
)

func readFile(path string) ([]byte, error) {
	path, err := homedir.Expand(path)
	if err != nil {
		errOut := fmt.Errorf("error while expanding config file path %s: %s", path, err)
		return []byte{}, errOut
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		err = fmt.Errorf("file %s does not exist", path)
		return []byte{}, err
	}

	data, err := ioutil.ReadFile(path)
	if err != nil {
		err = fmt.Errorf("error while reading %s: %s", path, err.Error())
		return data, err
	}

	return data, err
}

func readHypertext(url, body, method string, headers map[string]string) ([]byte, int, error) {
	client := &http.Client{}
	req, err := http.NewRequest(method, url, bytes.NewBufferString(body))
	if err != nil {
		err = fmt.Errorf("error while creating request: %s", err.Error())
		return []byte{}, 0, err
	}
	for k, v := range headers {
		req.Header.Add(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		err = fmt.Errorf("error while fetching from %s: %s", url, err.Error())
		return []byte{}, 0, err
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		err = fmt.Errorf("error while reading body of %s: %s", url, err.Error())
		return data, resp.StatusCode, err
	}

	return data, resp.StatusCode, nil
}

func readS3(bucket, object string) ([]byte, error) {
	awscfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		err = fmt.Errorf("error while creating s3 client to read %s from %s: %s", object, bucket, err.Error())
		return []byte{}, err
	}

	s3Client := s3.NewFromConfig(awscfg)
	input := &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(object),
	}

	result, err := s3Client.GetObject(context.TODO(), input)
	if err != nil {
		err = fmt.Errorf("error while reading object %s from %s: %s", object, bucket, err.Error())
		return []byte{}, err
	}
	defer result.Body.Close()
	data, err := ioutil.ReadAll(result.Body)
	if err != nil {
		err = fmt.Errorf("error while reading body of %s from %s: %s", object, bucket, err.Error())
		return data, err
	}

	return data, nil
}
