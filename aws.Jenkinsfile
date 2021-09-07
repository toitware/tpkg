pipeline {
    agent {
        kubernetes {
            yamlFile 'Jenkins.pod.yaml'
        }
    }

    environment {
        // See "Upload" step on the buildbot to find a newer version.
        TOIT_FIRMWARE_VERSION = "v1.3.0-pre.38+1f3c248cd"
    }

    stages {
        stage("Download") {
            steps {
                container('tpkg') {
                withCredentials([[$class: 'FileBinding', credentialsId: 'gcloud-service-auth', variable: 'GOOGLE_APPLICATION_CREDENTIALS']]) {
                    sh 'gcloud auth activate-service-account --key-file=$GOOGLE_APPLICATION_CREDENTIALS'
                    sh "gcloud config set project infrastructure-220307"
                    sh 'gsutil cp gs://toit-binaries/$TOIT_FIRMWARE_VERSION/sdk/$TOIT_FIRMWARE_VERSION.tar linux_firmware.tar'
                    sh 'gsutil cp gs://toit-archive/toit-devkit/darwin/$TOIT_FIRMWARE_VERSION.tgz darwin_sdk.tgz'
                    sh 'gsutil cp gs://toit-archive/toit-devkit/windows/$TOIT_FIRMWARE_VERSION.tgz windows_sdk.tgz'
                    stash name: 'linux_firmware', includes: 'linux_firmware.tar'
                    stash name: 'windows_sdk', includes: 'windows_sdk.tgz'
                    stash name: 'darwin_sdk', includes: 'darwin_sdk.tgz'
                }
                }
            }
        }
        stage("Build Windows") {
            steps {
                container('tpkg') {
                sh 'make go_dependencies'
                sh 'GOOS=windows make tpkg'
                sh 'mv build/tpkg build/tpkg.exe'
                stash name: 'win_tpkg', includes: 'build/tpkg.exe'
                }
            }
        }
        stage("Test") {
            parallel {

                stage("Windows") {
                    agent {
                        label 'windows'
                    }
                    stages {
                        stage("setup") {
                            steps {
                                unstash 'windows_sdk'
                                bat "mkdir test-tools"
                                bat "tar x -zf windows_sdk.tgz -C test-tools"
                                bat "go get -u github.com/jstemmer/go-junit-report"
                                unstash "win_tpkg"
                            }
                        }

                        stage("test") {
                            environment {
                                TPKG_PATH="${env.WORKSPACE}\\build\\tpkg.exe"
                                TOITLSP_PATH="${env.WORKSPACE}\\test-tools\\toitlsp.exe"
                                TOITC_PATH="${env.WORKSPACE}\\test-tools\\toitc.exe"
                            }
                            steps {
                                bat "dir"
                                // TODO(florian): enable Windows tests.
                                bat "tedi test -v ./tests/..."
                                // bat "tedi test -v -cover -race -bench=. ./tests/... 2>&1 | go-junit-report > tests.xml"
                            }
                            // post {
                            //     always {
                            //        junit "tests.xml"
                            //    }
                            // }
                        }
                    }
                }
            }
        }
    }
}
