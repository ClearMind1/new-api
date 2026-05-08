pipeline {
    agent any

    parameters {
        string(
            name: 'IMAGE_TAG',
            defaultValue: '',
            description: 'Required. Image tag, e.g. v0.10.8-alpha.3 or test-001. Image will be tagged as clearmind1/new-api:<IMAGE_TAG>-amd64.',
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
        stage('Validate Params') {
            steps {
                script {
                    if (!params.IMAGE_TAG?.trim()) {
                        error 'IMAGE_TAG parameter is required (e.g., v0.10.8-alpha.3)'
                    }
                    echo "Building tag: ${params.IMAGE_TAG} for ${env.ARCH}"
                }
            }
        }

        stage('Prepare VERSION') {
            steps {
                sh '''
                    echo "${IMAGE_TAG}" > VERSION
                    cat VERSION
                '''
            }
        }

        stage('Build Image') {
            steps {
                sh '''
                    docker build \
                      -t "${IMAGE_NAME}:${IMAGE_TAG}-${ARCH}" \
                      -t "${IMAGE_NAME}:latest-${ARCH}" \
                      .
                '''
            }
        }

        stage('Verify Image') {
            steps {
                sh '''
                    docker image inspect "${IMAGE_NAME}:${IMAGE_TAG}-${ARCH}" \
                      --format '{{.Id}} {{.Created}} {{.Size}}'
                    docker images "${IMAGE_NAME}" --filter reference="${IMAGE_NAME}:${IMAGE_TAG}-${ARCH}"
                '''
            }
        }
    }

    post {
        always {
            sh 'rm -f VERSION || true'
        }
        success {
            echo "Built ${env.IMAGE_NAME}:${params.IMAGE_TAG}-${env.ARCH}"
        }
        failure {
            echo "Build failed for tag ${params.IMAGE_TAG}"
        }
    }
}
