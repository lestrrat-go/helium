<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">

  <!-- stream-available() on a supplied stream -->
  
  <xsl:param name="uri" as="xs:string"/>
  
  <xsl:template name="main">
    <out><xsl:value-of select="stream-available($uri)"/></out>
  </xsl:template>

</xsl:stylesheet>
