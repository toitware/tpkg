pipeline {
    agent {
        kubernetes {
            yamlFile 'Jenkins.pod.yaml'
        }
    }

    environment {
        // See "Upload" step on the buildbot to find a newer version.
        TOIT_FIRMWARE_VERSION = "v1.3.0-pre.29+ed090adcb"
    }

    stages {
        stage("Download") {
            agent {
                kubernetes {
                    yamlFile 'Jenkins.pod.yaml'
                    defaultContainer 'tpkg'
                }
            }
            options {
                skipDefaultCheckout()
            }
            steps {
                withCredentials([[$class: 'FileBinding', credentialsId: 'gcloud-service-auth', variable: 'GOOGLE_APPLICATION_CREDENTIALS']]) {
                    sh 'gcloud auth activate-service-account --key-file=$GOOGLE_APPLICATION_CREDENTIALS'
                    sh "gcloud config set project infrastructure-220307"
                    sh 'gsutil cp gs://toit-binaries/$TOIT_FIRMWARE_VERSION/sdk/$TOIT_FIRMWARE_VERSION.tar linux.tar'
                    sh 'gsutil cp gs://toit-archive/toit-devkit/darwin/$TOIT_FIRMWARE_VERSION.tgz darwin.tgz'
                    sh 'gsutil cp gs://toit-archive/toit-devkit/windows/$TOIT_FIRMWARE_VERSION.tgz windows.tgz'
                    stash name: 'linux', includes: 'linux.tar'
                    stash name: 'windows', includes: 'windows.tgz'
                    stash name: 'darwin', includes: 'darwin.tgz'
                }
            }
        }
        stage("Build Windows") {
            agent {
                kubernetes {
                    yamlFile 'Jenkins.pod.yaml'
                    defaultContainer 'tpkg'
                }
            }
            steps {
                sh 'make go_dependencies'
                sh 'GOOS=windows make tpkg'
                sh 'mv build/tpkg build/tpkg.exe'
                stash name: 'win_tpkg', includes: 'build/tpkg.exe'
            }
        }
        stage("Test") {
            parallel {
                stage("Linux") {
                    agent {
                        kubernetes {
                            yamlFile 'Jenkins.pod.yaml'
                            defaultContainer 'tpkg'
                        }
                    }
                    stages {
                        stage("setup") {
                            steps {
                                unstash 'linux'
                                sh "mkdir test-tools"
                                sh "tar x -f linux.tar -C test-tools"
                                sh "make go_dependencies"
                                sh "go get -u github.com/jstemmer/go-junit-report"
                                sh "make -j 10 tpkg"
                            }
                        }
                        stage("test") {
                            environment {
                                TPKG_PATH="${env.WORKSPACE}/build/tpkg"
                                TOITVM_PATH="${env.WORKSPACE}/test-tools/toitvm"
                            }
                            steps {
                                sh "tedi test -v -cover -bench=. ./tests/... 2>&1 | tee tests.out"
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
                stage("Mac") {
                    agent {
                        label 'macos'
                    }
                    stages {
                        stage("setup") {
                            steps {
                                unstash 'darwin'
                                sh "mkdir test-tools"
                                sh "tar x -zf darwin.tgz -C test-tools"
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
                                sh "tedi test -v -cover -bench=. ./tests/... 2>&1 | tee tests.out"
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
                                unstash 'windows'
                                bat "mkdir test-tools"
                                bat "tar x -zf windows.tgz -C test-tools"
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
                                // bat "tedi test -v ./tests/..."
                                // bat "tedi test -v -cover -bench=. ./tests/... 2>&1 | go-junit-report > tests.xml"
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
