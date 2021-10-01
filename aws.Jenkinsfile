def TOIT_VERSION
pipeline {
    agent {
        kubernetes {
            yamlFile 'Jenkins.pod.yaml'
        }
    }

    environment {
        // See "Upload" step on the buildbot to find a newer version.
        TOIT_FIRMWARE_VERSION = "v1.4.0-pre.6+baa6f3c99"
    }

    stages {
        stage("Download") {
            steps {
                container('tpkg') {
                    script {
                        TOIT_VERSION=sh(returnStdout: true, script: 'gitversion').trim()
                    }
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
                stage("Linux") {
                    stages {
                        stage("setup") {
                            steps {
                                container('tpkg') {
                                    unstash 'linux_firmware'
                                    sh "mkdir test-tools"
                                    sh "tar x -f linux_firmware.tar -C test-tools"
                                    sh "make go_dependencies"
                                    sh "go get -u github.com/jstemmer/go-junit-report"
                                    sh "make -j 10 tpkg"
                                }
                            }
                        }
                        stage("test") {
                            environment {
                                TPKG_PATH="${env.WORKSPACE}/build/tpkg"
                                TOITVM_PATH="${env.WORKSPACE}/test-tools/toitvm64"
                            }
                            steps {
                                container('tpkg') {
                                    sh "TEST_FLAGS='-race -bench=.' make test 2>&1 | tee tests.out"
                                    sh "cat tests.out | go-junit-report > tests.xml"
                                }
                            }
                            post {
                                always {
                                    junit "tests.xml"
                                }
                            }
                        }
                    }
                }
                stage("Mac") {
                    agent {
                        label 'macos'
                    }
                    stages {
                        stage("setup") {
                            steps {
                                unstash 'darwin_sdk'
                                sh "mkdir test-tools"
                                sh "tar x -zf darwin_sdk.tgz -C test-tools"
                                sh "make go_dependencies"
                                sh "go get -u github.com/jstemmer/go-junit-report"
                                sh "make -j 10 tpkg"
                            }
                        }
                        stage("test") {
                            environment {
                                TPKG_PATH="${env.WORKSPACE}/build/tpkg"
                                TOITLSP_PATH="${env.WORKSPACE}/test-tools/toitlsp"
                                TOITC_PATH="${env.WORKSPACE}/test-tools/toitc"
                            }
                            steps {
                                sh "TEST_FLAGS='-race -bench=.' make test 2>&1 | tee tests.out"
                                sh "cat tests.out | go-junit-report > tests.xml"
                            }
                            post {
                                always {
                                    junit "tests.xml"
                                }
                            }
                        }
                    }
                }
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
                                bat "go get -u github.com/jstroem/tedi/cmd/tedi"
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
                                bat "tedi test -cover -bench=. -v ./cmd/... 2>&1 | go-junit-report > sdk_test.xml"
                            }
                            post {
                                always {
                                    junit "sdk_test.xml"
                                    cleanWs(cleanWhenNotBuilt: false, notFailBuild: true)
                                }
                            }
                        }
                    }
                }
            }
        }

        stage("Build registry") {
            when {
                anyOf {
                    branch 'master'
                    branch 'main'
                    branch pattern: "release-v\\d+.\\d+", comparator: "REGEXP"
                    tag "v*"
                }
            }

            steps {
                container('tpkg') {
                    sh "GCLOUD_IMAGE_TAG=${TOIT_VERSION} make gcloud"
                }
            }
        }
    }
}
