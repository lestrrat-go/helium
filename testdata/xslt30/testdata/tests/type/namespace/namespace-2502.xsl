<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    version="2.0"
    xmlns:a="http://a.uri/"
    xmlns:b="http://b.uri/"
    exclude-result-prefixes="#all">

  <!-- test exclude-result-prefixes="#all" -->
  <?spec xslt#lre-namespaces?>
  
  <!-- should copy c.uri only -->
  
  <xsl:template match="doc">
    <out xmlns:c="http://c.uri/" xmlns:d="http://d.uri/" xmlns:a2="http://a.uri/" xsl:exclude-result-prefixes="d"/>
  </xsl:template>
 
</xsl:stylesheet>
