# kospell

>  비상업적 용도로만 사용할 수 있습니다.

[(주) 나라인포테크 맞춤법 검사기](https://nara-speller.co.kr/speller/) in golang

```
go get github.com/Alfex4936/kospell
```


```json
{
  "original": "너는나와 kafka 머고나서\r\n",
  "charCount": 18,
  "chunkCount": 1,
  "corrections": [
    {
      "idx": 0,
      "input": "너는나와 kafka 머고나서\r\n",
      "items": [
        {
          "start": 0,
          "end": 4,
          "origin": "너는나와",
          "suggest": [
            "너는 나와"
          ],
          "help": "관형사형 어미 뒤에 오는 말은 띄어 씁니다.\n\n(예) 데뷔할예정(×) -> 데뷔할 예정(○)\n잘시간(×) -> 잘 시간(○)\n좋은사람(×) -> 좋은 사람(○)\n한가한때(×) -> 한가한 때(○)\n이런식으로(×) -> 이런 식으로(○)\n그런점(×) -> 그런 점(○)"
        },
        {
          "start": 5,
          "end": 10,
          "origin": "kafka",
          "suggest": [
            "Kafka",
            "KAFA",
            "kaka",
            "KFKA"
          ],
          "help": "영어 단어를 잘못 표기하셨습니다.\n\n고유명사의 첫 글자나 문장을 시작할 때는 대문자로 써야 하고, 다른 글자나 일반 명사는 소문자로 적습니다. 첫 글자만 따서 만든 단어일 때는 모두 대문자로 씁니다.\n\n(예) Seoul\nappletiME -> time\n\n로마자를 입력할 때도 한글처럼 알파벳을 잘못 입력하여 오류를 범할 수 있으므로 주의해야 합니다.\n\n(1) 알파벳의 순서를 바꿔 쓰거나 다른 글자를 입력한 오류\nadn -> and\ntheone -> throne\n\n(2) 필요한 글자가 빠지거나 불필요한 글자를 더 입력한 오류\ncomitment -> commitment\nwellcome -> welcome"
        },
        {
          "start": 11,
          "end": 15,
          "origin": "머고나서",
          "suggest": [
            "머고 나서"
          ],
          "help": "띄어쓰기 오류입니다. 대치어를 참고하여 띄어 쓰도록 합니다."
        }
      ]
    }
  ],
  "errorCount": 3
}
```