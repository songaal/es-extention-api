FROM golang:1.16.5

# 필수 패키지 설치
RUN apt update -y
RUN apt install sudo vim curl net-tools -y

# 유저 추가 후 변경 (root 권한 포함)
RUN groupadd -g 1000 danawa
RUN useradd -r -u 1000 -g danawa danawa
RUN echo 'danawa ALL=(ALL) NOPASSWD:ALL' >> /etc/sudoers
USER danawa

COPY dist/application application
RUN sudo chown -R danawa:danawa application

RUN echo date > build.txt
