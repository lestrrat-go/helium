<?xml version="1.0"?>
<xsl:stylesheet
  xmlns:my="http://example.com/functions"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">
  
  <!-- test xsl:copy on-empty - empty text node -->
  
      
  <xsl:template match="/">
    <xsl:variable name="var1" as="node()"><xsl:value-of select="''"/></xsl:variable>
    <out>
      <xsl:where-populated>
        <xsl:copy select="$var1"/>
      </xsl:where-populated> 
      <xsl:on-empty select="my:node()"/> 
    </out>  
  </xsl:template>
  
  <xsl:function name="my:node">
    <foo />
  </xsl:function>

</xsl:stylesheet>