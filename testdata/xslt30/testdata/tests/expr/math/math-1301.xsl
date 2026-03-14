<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec fo#func-number?>
  <!-- Purpose: Test of is-a-number technique. -->
  
<xsl:template match="doc">
  <out>
    <xsl:for-each select="n">
      <xsl:choose>
        <xsl:when test="contains(string(number(.)),'NaN')">
          <xsl:value-of select="."/><xsl:text> is not a number
</xsl:text>
        </xsl:when>
        <xsl:otherwise>
          <xsl:value-of select="."/><xsl:text> is a number
</xsl:text>
        </xsl:otherwise>
      </xsl:choose>
    </xsl:for-each>
  </out>
</xsl:template>

</xsl:stylesheet>
