<bar:stylesheet xmlns:bar="http://www.w3.org/1999/XSL/Transform" version="2.0">
<?spec xslt#shallow-copy?>
  <bar:template match="*">
    <bar:copy>
      <bar:copy-of select="./@*"/>
      <bar:apply-templates/>
    </bar:copy>
  </bar:template>
</bar:stylesheet>
