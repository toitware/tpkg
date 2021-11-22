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
        stage("Test") {
            parallel {
                stage("Linux") {
                    stages {
                        stage("setup") {
                            steps {
                                container('tpkg') {
                                    sh "make go_dependencies"
                                    sh "go get -u github.com/jstemmer/go-junit-report"
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
