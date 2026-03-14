<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

<!-- test xsl:number level="any" with a count pattern that matches the document root node -->
<?spec xslt#number?>
  <xsl:template match="/">
    <z><xsl:apply-templates select="processing-instruction() | comment() | //book/@style"/></z>
  </xsl:template>

  <xsl:template match="node() | @*">
    <n><xsl:number level="any" count="node() | / | @*"/></n>
  </xsl:template>
</xsl:stylesheet>
