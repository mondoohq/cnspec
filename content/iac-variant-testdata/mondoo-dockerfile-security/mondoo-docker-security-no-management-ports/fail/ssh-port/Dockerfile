FROM ubuntu:24.04
RUN apt-get update && apt-get install -y --no-install-recommends openssh-server && rm -rf /var/lib/apt/lists/*
EXPOSE 22
CMD ["/usr/sbin/sshd", "-D"]
