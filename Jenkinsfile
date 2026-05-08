pipeline {
    agent any

    parameters {
        string(
            name: 'IMAGE_TAG',
            defaultValue: '',
            description: 'Optional. Leave empty to auto-generate as <yyyymmdd>-<short-sha> (matches docker-image-nightly.yml). Otherwise overrides the image tag suffix. Image will be tagged as clearmind1/new-api:<TAG>-amd64.',
            trim: true
        )
    }

    environment {
        IMAGE_NAME = 'clearmind1/new-api'
        ARCH       = 'amd64'
    }

    options {
        timestamps()
        timeout(time: 60, unit: 'MINUTES')
        disableConcurrentBuilds()
    }

    stages {
        stage('Resolve Tag') {
            steps {
                script {
                    def tag = params.IMAGE_TAG?.trim()
                    if (tag) {
                        echo "Using provided IMAGE_TAG: ${tag}"
                    } else {
                        def date = sh(script: "date +'%Y%m%d'", returnStdout: true).trim()
                        def sha  = sh(script: 'git rev-parse --short HEAD', returnStdout: true).trim()
                        tag = "${date}-${sha}"
                        echo "IMAGE_TAG not provided, auto-generated: ${tag}"
                    }
                    env.RESOLVED_TAG = tag
                    currentBuild.displayName = "#${env.BUILD_NUMBER} ${tag}"
                }
            }
        }

        stage('Prepare VERSION') {
            steps {
                sh '''
                    echo "${RESOLVED_TAG}" > VERSION
                    cat VERSION
                '''
            }
        }

        stage('Build Image') {
            steps {
                sh '''
                    docker build \
                      -t "${IMAGE_NAME}:${RESOLVED_TAG}-${ARCH}" \
                      -t "${IMAGE_NAME}:latest-${ARCH}" \
                      .
                '''
            }
        }

        stage('Verify Image') {
            steps {
                sh '''
                    docker image inspect "${IMAGE_NAME}:${RESOLVED_TAG}-${ARCH}" \
                      --format '{{.Id}} {{.Created}} {{.Size}}'
                    docker images "${IMAGE_NAME}" --filter reference="${IMAGE_NAME}:${RESOLVED_TAG}-${ARCH}"
                '''
            }
        }
    }

    post {
        always {
            sh 'rm -f VERSION || true'
        }
        success {
            echo "Built ${env.IMAGE_NAME}:${env.RESOLVED_TAG}-${env.ARCH}"
        }
        failure {
            echo "Build failed for tag ${env.RESOLVED_TAG ?: '(unresolved)'}"
        }
    }
}
