<xsl:stylesheet
  version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">

    <xsl:global-context-item use="absent"/>

    <xsl:template name="xsl:initial-template">
      <xsl:copy-of select="$var"/>
    </xsl:template>
  
  <xsl:variable name="var">
    <out>
      <xsl:copy>
        <in/>
      </xsl:copy>
    </out>
  </xsl:variable>
  
  </xsl:stylesheet>