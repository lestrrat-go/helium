<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">

  <!-- current-output-uri() in a global variable -->

  <xsl:variable name="v">
    <xsl:choose>
      <xsl:when test="empty(current-output-uri())"><ok/></xsl:when>
      <xsl:otherwise><wrong result="{current-output-uri()}"/></xsl:otherwise>
    </xsl:choose>
  </xsl:variable>
  
  <xsl:template name="main">
    <xsl:copy-of select="$v"/>
  </xsl:template>

</xsl:stylesheet>
