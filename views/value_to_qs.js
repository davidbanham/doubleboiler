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

{{ define "value_to_qs_child" }}
<script>
(function() {
  var parent = document.getElementById("{{.ParentID}}")
  parent.addEventListener("change", function(ev) {
    var searchParams = new URLSearchParams(window.location.search);
    searchParams.set('{{.Key}}', ev.target.value);
    window.location.search = searchParams.toString();
  });
})();
</script>
{{ end }}
