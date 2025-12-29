"""
URL configuration for async_service project.
"""
from django.urls import path
from calculator import views

urlpatterns = [
    path('api/calculate/', views.calculate_view, name='calculate'),
]
