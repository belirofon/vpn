abstract class HttpClient {
  Future<HttpResponse> get(String url, {Map<String, String>? headers});
  Future<HttpResponse> post(String url,
      {dynamic data, Map<String, String>? headers});
  Future<HttpResponse> put(String url,
      {dynamic data, Map<String, String>? headers});
  Future<HttpResponse> delete(String url, {Map<String, String>? headers});
  void updateBaseUrl(String url);
}

class HttpResponse {
  final int statusCode;
  final Map<String, dynamic>? data;

  const HttpResponse({required this.statusCode, this.data});
}
