{{ define "value_to_qs" }}
var searchParams = new URLSearchParams(window.location.search);
searchParams.set('{{.}}', this.value);
window.location.search = searchParams.toString();
{{ end }}

{{ define "value_to_qs_toggle" }}
var searchParams = new URLSearchParams(window.location.search);
searchParams.set('{{.}}', this.checked);
window.location.search = searchParams.toString();
{{ end }}
