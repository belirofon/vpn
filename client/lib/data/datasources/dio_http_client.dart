import 'package:dio/dio.dart';
import '../../domain/repositories/http_client.dart' as domain;

class DioHttpClient implements domain.HttpClient {
  late final Dio _dio;

  DioHttpClient({String baseUrl = 'http://localhost:8080'}) {
    _dio = Dio(
      BaseOptions(
        baseUrl: baseUrl,
        connectTimeout: const Duration(seconds: 10),
        receiveTimeout: const Duration(seconds: 15),
        sendTimeout: const Duration(seconds: 10),
        headers: {
          'Accept': 'application/json',
          'Content-Type': 'application/json',
        },
      ),
    );
  }

  @override
  void updateBaseUrl(String url) {
    _dio.options.baseUrl = url;
  }

  @override
  Future<domain.HttpResponse> get(
    String url, {
    Map<String, String>? headers,
  }) async {
    try {
      final response = await _dio.get(
        url,
        options: Options(headers: headers),
      );
      return domain.HttpResponse(
        statusCode: response.statusCode ?? 500,
        data: response.data is Map<String, dynamic>
            ? response.data as Map<String, dynamic>
            : null,
      );
    } on DioException {
      rethrow;
    }
  }

  @override
  Future<domain.HttpResponse> post(
    String url, {
    dynamic data,
    Map<String, String>? headers,
  }) async {
    try {
      final response = await _dio.post(
        url,
        data: data,
        options: Options(headers: headers),
      );
      return domain.HttpResponse(
        statusCode: response.statusCode ?? 500,
        data: response.data is Map<String, dynamic>
            ? response.data as Map<String, dynamic>
            : null,
      );
    } on DioException {
      rethrow;
    }
  }

  @override
  Future<domain.HttpResponse> delete(
    String url, {
    Map<String, String>? headers,
  }) async {
    try {
      final response = await _dio.delete(
        url,
        options: Options(headers: headers),
      );
      return domain.HttpResponse(
        statusCode: response.statusCode ?? 500,
        data: response.data is Map<String, dynamic>
            ? response.data as Map<String, dynamic>
            : null,
      );
    } on DioException {
      rethrow;
    }
  }

  @override
  Future<domain.HttpResponse> put(
    String url, {
    dynamic data,
    Map<String, String>? headers,
  }) async {
    try {
      final response = await _dio.put(
        url,
        data: data,
        options: Options(headers: headers),
      );
      return domain.HttpResponse(
        statusCode: response.statusCode ?? 500,
        data: response.data is Map<String, dynamic>
            ? response.data as Map<String, dynamic>
            : null,
      );
    } on DioException {
      rethrow;
    }
  }
}
