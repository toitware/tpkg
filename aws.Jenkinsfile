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
        stage("setup") {
            steps {
                sh "make go_dependencies"
            }
        }

        stage("Testing Linux") {
            agent {
                kubernetes {
                    yamlFile 'Jenkins.pod.yaml'
                    defaultContainer 'tpkg'
                }
            }
            steps {
                withCredentials([[$class: 'FileBinding', credentialsId: 'gcloud-service-auth', variable: 'GOOGLE_APPLICATION_CREDENTIALS']]) {
                    sh "gcloud auth activate-service-account --key-file=${GOOGLE_APPLICATION_CREDENTIALS}"
                    sh "gcloud config set project infrastructure-220307"
                    sh "./jenkins/test.sh"
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
