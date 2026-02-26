#!/bin/sh
set -e

ARGS="-p ${PORT:-8080} -mode ${MODE:-nara} -lang ${DICT_LANG:-ko}"

case "${MODE:-nara}" in
  hunspell)
    if [ -n "${DICT_DIR}" ]; then
      ARGS="$ARGS -dict ${DICT_DIR}"
    fi
    ;;
  openai)
    if [ -n "${OPENAI_API_KEY}" ]; then
      ARGS="$ARGS -llm-key ${OPENAI_API_KEY}"
    fi
    # LLM_MODEL takes priority; fall back to OPENAI_MODEL for convenience
    _MODEL="${LLM_MODEL:-${OPENAI_MODEL:-}}"
    if [ -n "${_MODEL}" ]; then
      ARGS="$ARGS -llm-model ${_MODEL}"
    fi
    if [ -n "${LLM_BASE_URL}" ]; then
      ARGS="$ARGS -llm-url ${LLM_BASE_URL}"
    fi
    ;;
esac

exec kospell-server $ARGS
