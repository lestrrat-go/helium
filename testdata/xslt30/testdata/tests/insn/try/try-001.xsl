<?xml version="1.0"?> 

<!-- try001 (was saxon081) -->

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
>

<!-- Test xsl:try -->


<xsl:template name="main">
  <o>
    <xsl:try>
      <xsl:value-of select="1 div 0"/>
      <xsl:catch>
        <xsl:text>divide by zero</xsl:text>
      </xsl:catch>
    </xsl:try>
  </o>
</xsl:template>


</xsl:stylesheet>
