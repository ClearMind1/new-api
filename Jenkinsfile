pipeline {
    agent any

    parameters {
        string(
            name: 'IMAGE_TAG',
            defaultValue: '',
            description: 'Optional. Leave empty to auto-generate as <yyyymmdd>-<short-sha> (matches docker-image-nightly.yml). Otherwise overrides the image tag suffix. Image will be tagged as clearmind1/new-api:<TAG>-amd64.',
            trim: true
        )
        string(
            name: 'REMOTE_HOST',
            defaultValue: '',
            description: 'Remote server IP or hostname for deployment. Leave empty to skip deploy.'
        )
        booleanParam(
            name: 'PUSH',
            defaultValue: true,
            description: 'When true, push the built image to Docker Hub (clearmind1/new-api) using the dockerhub-credentials credential ID. Uncheck for build-only debugging.'
        )
        booleanParam(
            name: 'DEPLOY',
            defaultValue: true,
            description: 'When true and PUSH is also true, deploy the new image to the remote server via SSH after push.'
        )
    }

    environment {
        IMAGE_NAME     = 'clearmind1/new-api'
        ARCH           = 'amd64'
        REMOTE_USER    = 'root'
        REMOTE_DEPLOY  = '/opt/new-api/deploy.sh'
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

        stage('Login to Docker Hub') {
            when { expression { return params.PUSH } }
            steps {
                withCredentials([usernamePassword(
                    credentialsId: 'dockerhub-credentials',
                    usernameVariable: 'DOCKERHUB_USER',
                    passwordVariable: 'DOCKERHUB_PASS'
                )]) {
                    sh '''
                        set +x
                        echo "$DOCKERHUB_PASS" | docker login -u "$DOCKERHUB_USER" --password-stdin
                    '''
                }
            }
        }

        stage('Push Image') {
            when { expression { return params.PUSH } }
            steps {
                sh '''
                    docker push "${IMAGE_NAME}:${RESOLVED_TAG}-${ARCH}"
                    docker push "${IMAGE_NAME}:latest-${ARCH}"
                '''
            }
        }

        stage('Deploy') {
            when {
                allOf {
                    expression { return params.PUSH }
                    expression { return params.DEPLOY }
                    expression { return params.REMOTE_HOST?.trim() }
                }
            }
            steps {
                script {
                    env.REMOTE_HOST = params.REMOTE_HOST.trim()
                }
                sshagent(credentials: ['remote-server-ssh-hk']) {
                    sh '''
                        ssh -o StrictHostKeyChecking=no \
                            ${REMOTE_USER}@${REMOTE_HOST} \
                            "${REMOTE_DEPLOY} ${RESOLVED_TAG}"
                    '''
                }
            }
        }
    }

    post {
        always {
            sh '''
                docker logout || true
                rm -f VERSION || true
            '''
        }
        success {
            script {
                echo "Built ${env.IMAGE_NAME}:${env.RESOLVED_TAG}-${env.ARCH}"
                if (params.PUSH) {
                    echo "Pushed ${env.IMAGE_NAME}:latest-${env.ARCH}"
                } else {
                    echo "PUSH disabled, image kept local only"
                }
            }
        }
        failure {
            echo "Build failed for tag ${env.RESOLVED_TAG ?: '(unresolved)'}"
        }
    }
}
