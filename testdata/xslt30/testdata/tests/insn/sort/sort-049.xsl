<?xml version="1.0"?>
<?spec xslt#sorting?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- FileName: SORT39 -->
  <!-- Document: http://www.w3.org/TR/xslt -->
  <!-- DocVersion: 19991116 -->
  <!-- Section: 10 -->
    <!-- Purpose: Try to set lang from an expression. -->

<xsl:output method="xml" indent="no"/>
<xsl:template match="data">
  <xsl:param name="language" select="'en'"/>
  <xsl:param name="country" select="'US'"/>
  <out><xsl:text>
  </xsl:text>
    <xsl:for-each select="row">
      <xsl:sort lang="{concat($language,'-',$country)}"/>
      <xsl:copy-of select="."/><xsl:text>
  </xsl:text>
    </xsl:for-each>
  </out>
</xsl:template>

</xsl:stylesheet>