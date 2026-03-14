<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">

<!-- test xsl:copy on-empty - empty document node-->

    
<xsl:template match="/">
  <xsl:variable name="var1"><rtf p="12">abc</rtf></xsl:variable>
  <out>
    <xsl:copy>
      <xsl:document>
        <xsl:sequence select=".//non-existent"/>
        <xsl:on-empty select="$var1//*"/>
      </xsl:document>  
    </xsl:copy>  
  </out>  
</xsl:template>

</xsl:stylesheet>