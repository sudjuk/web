"""
Простой тест для проверки работы Django эндпоинта
"""
import httpx
import json
import asyncio

async def test():
    url = "http://localhost:8000/api/calculate/"
    payload = {
        "observationId": 1,
        "callbackUrl": "http://localhost:8080/api/internal/asteroid-observations/1/calc-result",
        "token": "INTERNAL123"
    }

    print(f"Отправка запроса на {url}")
    print(f"Payload: {json.dumps(payload, indent=2)}")

    try:
        async with httpx.AsyncClient(timeout=5.0) as client:
            response = await client.post(url, json=payload)
            print(f"\nСтатус ответа: {response.status_code}")
            print(f"Ответ: {response.text}")
    except httpx.ConnectError:
        print("\nОШИБКА: Не удалось подключиться к серверу. Убедитесь, что Django сервис запущен на порту 8000")
    except Exception as e:
        print(f"\nОШИБКА: {e}")

if __name__ == "__main__":
    asyncio.run(test())
