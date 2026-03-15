<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">

  <!-- context position is not defined within xsl:on-completion -->

  <xsl:template name="main">
    <out>
      <xsl:iterate select="1 to 10">        
        <xsl:on-completion>
          <xsl:value-of select="position()"/>
        </xsl:on-completion>
        <xsl:value-of select="position()"/>
      </xsl:iterate>
    </out>
  </xsl:template>


</xsl:stylesheet>
