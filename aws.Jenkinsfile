def TOIT_VERSION
pipeline {
    agent {
        kubernetes {
            yamlFile 'Jenkins.pod.yaml'
        }
    }

    stages {
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
