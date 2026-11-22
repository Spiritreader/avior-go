# Description comparison adapter

`compare.py` is a thin runtime adapter around the shared `movie_nfo_lib` package. It does not contain a prompt or model mapping. The shared library resolves the configured model-specific prompt and Ollama instance.

Input is one JSON object on stdin:

```json
{"left":"description A","right":"description B","threshold":0.7}
```

Output is one JSON object on stdout:

```json
{"score":0.83,"same_content":true,"explanation":"..."}
```

`same_content` is true only when `score > threshold`. The caller supplies and validates the threshold.

The runtime sets `PYTHONPATH` to the checked-out `movie_nfo_lib` package and provides its `OLLAMA_*` environment variables. Errors produce a non-zero exit status and a short stderr message.
