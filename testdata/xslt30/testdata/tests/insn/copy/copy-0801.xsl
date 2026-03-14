<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#shallow-copy?>   
    <!-- Purpose: Test for-each inside xsl:copy -->                

<xsl:template match="/">
  <out>
    <xsl:copy>
      <xsl:for-each select="OL/OL/LI">
        <xsl:value-of select="."/>
      </xsl:for-each>
    </xsl:copy>
  </out>
</xsl:template>

</xsl:stylesheet>
