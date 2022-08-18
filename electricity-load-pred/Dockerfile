FROM pytorch/pytorch:latest
COPY load-pred.py /src/
COPY config /src/
RUN pip3 install kubernetes -i https://mirrors.aliyun.com/pypi/simple
WORKDIR /src
CMD ["python3", "load-pred.py"]
