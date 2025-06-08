package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/BlackBN/KubenetesDemo/etcd-operator/internal/file"
	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"

	//"go.etcd.io/etcd/clientv3/snapshot" // 快照功能
	clientv3 "go.etcd.io/etcd/client/v3"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func logErr(log logr.Logger, err error, message string) error {
	log.Error(err, message)
	return fmt.Errorf("%s,%v", message, err)
}

func main() {
	var (
		backupTempDir      string
		etcdURL            string
		dialTimeoutSeconds int64
		timeoutSeconds     int64
	)

	flag.StringVar(&backupTempDir, "backup-tmp-dir", os.TempDir(), "the dir to temp place backup etcd cluster")
	flag.StringVar(&etcdURL, "etcd-url", "", "url for the backup etcd")
	flag.Int64Var(&dialTimeoutSeconds, "dial-timeout-seconds", 5, "dialing timeout for the backup etcd")
	flag.Int64Var(&timeoutSeconds, "timeout-seconds", 60, "timeout for the backup etcd")

	timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(timeoutSeconds))
	defer cancel()
	zapLogger := zap.NewRaw(zap.UseDevMode(true))
	ctrl.SetLogger(zapr.NewLogger(zapLogger))
	log := ctrl.Log.WithName("backup")

	log.Info("init etcd client and snapshot to local dir")
	localPath := filepath.Join(backupTempDir, "snapshot.db")
	//etcdManager := snapshot.NewV3(zapLogger)
	// if err := etcdManager.Save(timeoutCtx, clientv3.Config{
	// 	Endpoints: []string{
	// 		etcdURL,
	// 	},
	// 	DialTimeout: time.Second * time.Duration(dialTimeoutSeconds),
	// }, localPath); err != nil {
	//
	// }

	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{etcdURL}, //如果是集群，就在后面加所有的节点[]string{"localhost:2379", "localhost:22379", "localhost:32379"},
		DialTimeout: time.Second * time.Duration(dialTimeoutSeconds),
	})
	if err != nil {
		panic(logErr(log, err, "failed to get etcd snapshot data"))
	}
	defer cli.Close()

	//数据保存成功后，上传
	endpoint := "play.min.io"
	accessKeyID := "Q3AM3UQ867SPQQA43P2F"
	secretAccessKey := "zuf+tfteSlswRu7BJ86wekitnifILbZam1KYY3TG"
	useSSL := true
	s3Uploader := file.NewS3Uploader(endpoint, accessKeyID, secretAccessKey, useSSL)

	log.Info("begin uploading snapshot ....")
	size, err := s3Uploader.Upload(timeoutCtx, localPath)
	if err != nil {
		panic(logErr(log, err, "failed to upload backup etcd"))
	}
	log.WithValues("upload-size", size).Info("Backup success")
}
