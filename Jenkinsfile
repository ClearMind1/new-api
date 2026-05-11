pipeline {
    agent any

    parameters {
        string(
            name: 'IMAGE_TAG',
            defaultValue: '',
            description: '可选。留空将自动生成 <yyyymmdd>-<short-sha> 格式的标签（与 docker-image-nightly.yml 保持一致）。填写则覆盖默认标签后缀。镜像最终标签为 clearmind1/new-api:<TAG>-amd64。',
            trim: true
        )
        booleanParam(
            name: 'PUSH',
            defaultValue: true,
            description: '勾选时，使用 dockerhub-credentials 凭据将构建好的镜像推送到 Docker Hub (clearmind1/new-api)。仅本地构建调试时取消勾选。'
        )
        booleanParam(
            name: 'DEPLOY',
            defaultValue: true,
            description: '勾选时（且 PUSH 也勾选），推送完成后通过 SSH 将新镜像部署到远程服务器。主机/端口从 Jenkins 凭据 deploy-remote-host-hk 与 deploy-remote-port-hk 读取。'
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
                }
            }
            steps {
                withCredentials([
                    string(credentialsId: 'deploy-remote-host-hk', variable: 'REMOTE_HOST'),
                    string(credentialsId: 'deploy-remote-port-hk', variable: 'REMOTE_PORT')
                ]) {
                    sshagent(credentials: ['remote-server-ssh-hk']) {
                        sh '''
                            ssh -o StrictHostKeyChecking=no \
                                -p ${REMOTE_PORT} \
                                ${REMOTE_USER}@${REMOTE_HOST} \
                                "${REMOTE_DEPLOY} ${RESOLVED_TAG}"
                        '''
                    }
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
