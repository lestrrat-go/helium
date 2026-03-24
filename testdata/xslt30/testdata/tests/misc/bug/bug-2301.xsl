<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
xmlns:a="http://uri1.uri/">

<?spec xslt#patterns?>
<!-- BUG: Saxon 5.3.2/025 -->
<!-- DESCRIPTION: When an attribute node is matched against the pattern "@prefix:*", a null
    pointer exception occurs. -->
    
  <xsl:template match="foo">
    <out><xsl:apply-templates select="@*"/></out>
  </xsl:template>
  
  <xsl:template match="@a:*">
    <a>prefix matched</a>
  </xsl:template>

  <xsl:template match="@*">
    <b>default matched</b>
  </xsl:template>
  
</xsl:stylesheet>

