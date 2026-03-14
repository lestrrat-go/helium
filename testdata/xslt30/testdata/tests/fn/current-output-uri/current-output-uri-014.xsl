<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">

  <!-- current-output-uri() initial value when not originally set in the test should be empty -->
  
  <xsl:template name="xsl:initial-template">
    <out><xsl:value-of select="'base-output-uri: ' || current-output-uri()"/></out>
  </xsl:template>

</xsl:stylesheet>
