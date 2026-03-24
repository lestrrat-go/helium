<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- Purpose: Set a variable to a boolean, and show it being used in xsl:if -->
  
<xsl:template match="doc">
  <xsl:variable name="partest" select="contains('foo','o')"/>
  <out>
    <xsl:if test="$partest">
      <xsl:text>Success</xsl:text>
    </xsl:if>
  </out>
</xsl:template>

</xsl:stylesheet>
