<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
<!-- Failure to convert empty string to number -->
  <xsl:template match="/">
    <xsl:variable name="doc" as="element()+"><a/><b/><c/></xsl:variable>   
    <xsl:sequence select="$doc = 3"/>
  </xsl:template>
</xsl:stylesheet>