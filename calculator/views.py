"""
Views for async calculation service.
"""
import asyncio
import json
import logging
import random
from datetime import datetime
from math import sin, cos, atan2, sqrt, radians, degrees

import httpx
from django.http import JsonResponse
from django.views.decorators.csrf import csrf_exempt
from django.views.decorators.http import require_http_methods

# Настройка логирования
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)

# Используем threading для запуска асинхронной функции в отдельном потоке

def angular_distance(ra1_rad, dec1_rad, ra2_rad, dec2_rad):
    """
    Вычисляет угловое расстояние между двумя точками на сфере
    по формуле гаверсинуса.
    
    α = 2 × atan2(√a, √(1-a))
    a = sin²(ΔDEC/2) + cos(DEC₁) × cos(DEC₂) × sin²(ΔRA/2)
    """
    delta_ra = ra2_rad - ra1_rad
    delta_dec = dec2_rad - dec1_rad
    
    a = sin(delta_dec / 2) ** 2 + cos(dec1_rad) * cos(dec2_rad) * sin(delta_ra / 2) ** 2
    
    # Защита от выхода за границы [-1, 1]
    if a > 1:
        a = 1
    elif a < 0:
        a = 0
    
    alpha = 2 * atan2(sqrt(a), sqrt(1 - a))
    return alpha


def calculate_asteroid_distance(points):
    """
    Вычисляет расстояние до астероида по формуле из calculation.txt
    
    Формула: distance = V × Δt / α
    где V = 25000 м/с (линейная скорость)
    α - угловое расстояние между наблюдениями (радианы)
    Δt - время между наблюдениями (секунды)
    
    Для всех пар наблюдений вычисляется расстояние, затем берётся медиана.
    """
    if not points or len(points) < 2:
        logger.warning("Not enough points for calculation")
        return 0.0
    
    # Парсим даты и сортируем по времени
    parsed_points = []
    for p in points:
        date_str = p.get('date', '').strip()
        if not date_str:
            continue
        
        # Пробуем разные форматы дат
        try:
            # DD.MM.YYYY
            dt = datetime.strptime(date_str, "%d.%m.%Y")
        except ValueError:
            try:
                # YYYY-MM-DD
                dt = datetime.strptime(date_str, "%Y-%m-%d")
            except ValueError:
                logger.warning(f"Could not parse date: {date_str}")
                continue
        
        parsed_points.append({
            'ra': p.get('ra', 0.0),
            'dec': p.get('dec', 0.0),
            'time': dt
        })
    
    if len(parsed_points) < 2:
        logger.warning("Not enough valid points after parsing dates")
        return 0.0
    
    # Сортируем по времени
    parsed_points.sort(key=lambda x: x['time'])
    
    # Вычисляем расстояния для всех пар
    LINEAR_SPEED = 25000.0  # м/с
    distances_km = []
    
    for i in range(len(parsed_points) - 1):
        for j in range(i + 1, len(parsed_points)):
            p1 = parsed_points[i]
            p2 = parsed_points[j]
            
            # Время между наблюдениями в секундах
            dt_seconds = (p2['time'] - p1['time']).total_seconds()
            if dt_seconds <= 0:
                continue
            
            # Угловое расстояние в радианах
            alpha = angular_distance(
                radians(p1['ra']), radians(p1['dec']),
                radians(p2['ra']), radians(p2['dec'])
            )
            
            if alpha <= 0:
                continue
            
            # Расстояние = V × Δt / α (в метрах)
            distance_meters = LINEAR_SPEED * dt_seconds / alpha
            
            if distance_meters > 0 and not (distance_meters == float('inf') or distance_meters != distance_meters):
                distances_km.append(distance_meters / 1000.0)  # конвертируем в км
    
    if not distances_km:
        logger.warning("No valid distances calculated")
        return 0.0
    
    # Возвращаем медиану
    distances_km.sort()
    n = len(distances_km)
    if n % 2 == 1:
        return distances_km[n // 2]
    else:
        return (distances_km[n // 2 - 1] + distances_km[n // 2]) / 2.0


@csrf_exempt
@require_http_methods(["POST"])
def calculate_view(request):
    """
    Асинхронный эндпоинт для расчёта расстояния до астероида.
    
    Принимает JSON:
    {
        "observationId": int,
        "callbackUrl": "http://...",
        "token": "INTERNAL123"
    }
    
    Выполняет расчёт с задержкой 5-10 секунд, затем отправляет результат
    обратно в основной сервис через callbackUrl.
    """
    try:
        data = json.loads(request.body)
        observation_id = data.get('observationId')
        callback_url = data.get('callbackUrl')
        token = data.get('token')
        
        if not observation_id or not callback_url or not token:
            return JsonResponse(
                {'error': 'Missing required fields: observationId, callbackUrl, token'},
                status=400
            )
        
        # Запускаем асинхронную задачу расчёта в фоне
        # Используем threading для запуска асинхронной функции в отдельном потоке
        import threading
        
        points = data.get('points', [])
        
        def run_async():
            loop = asyncio.new_event_loop()
            asyncio.set_event_loop(loop)
            loop.run_until_complete(perform_calculation(observation_id, callback_url, token, points))
            loop.close()
        
        thread = threading.Thread(target=run_async, daemon=True)
        thread.start()
        
        # Сразу возвращаем успешный ответ
        return JsonResponse({'status': 'accepted', 'observationId': observation_id}, status=202)
        
    except json.JSONDecodeError:
        return JsonResponse({'error': 'Invalid JSON'}, status=400)
    except Exception as e:
        logger.error(f"Error in calculate_view: {e}", exc_info=True)
        return JsonResponse({'error': 'Internal server error'}, status=500)


async def perform_calculation(observation_id: int, callback_url: str, token: str, points: list = None):
    """
    Выполняет расчёт расстояния до астероида с задержкой 5-10 секунд,
    затем отправляет результат обратно в основной сервис.
    
    points: список точек наблюдения с полями ra, dec, date
    """
    try:
        # Задержка 5-10 секунд
        delay = random.uniform(5.0, 10.0)
        logger.info(f"Starting calculation for observation {observation_id}, delay: {delay:.2f} seconds")
        await asyncio.sleep(delay)
        logger.info(f"Delay finished, starting calculation for observation {observation_id}")
        
        # Выполняем реальный расчёт по формуле
        calculated_km = calculate_asteroid_distance(points)
        success = calculated_km > 0
        logger.info(f"Calculation completed for observation {observation_id}: {calculated_km:.2f} km, success: {success}")
        
        # Отправляем результат обратно в основной сервис
        result_payload = {
            'success': success,
            'value': calculated_km if success else 0.0
        }
        
        headers = {
            'Content-Type': 'application/json',
            'X-Internal-Token': token
        }
        
        async with httpx.AsyncClient(timeout=30.0) as client:
            response = await client.post(
                callback_url,
                json=result_payload,
                headers=headers
            )
            
            if response.status_code >= 200 and response.status_code < 300:
                logger.info(f"Successfully sent calculation result for observation {observation_id}")
            else:
                logger.error(
                    f"Failed to send calculation result for observation {observation_id}: "
                    f"status {response.status_code}, body: {response.text}"
                )
                
    except Exception as e:
        logger.error(
            f"Error in perform_calculation for observation {observation_id}: {e}",
            exc_info=True
        )
