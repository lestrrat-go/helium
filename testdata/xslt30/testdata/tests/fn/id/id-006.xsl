<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec fo#func-id?>
    <!-- Purpose: Test for id(list) and position() in a for-each loop -->

<xsl:output method="xml" encoding="UTF-8"/>

<xsl:template match="/">
  <out>
    <xsl:text>
</xsl:text>
    <xsl:for-each select="id('d b c')">
      <item>
        <xsl:value-of select="./@id"/>
        <xsl:text> is at position </xsl:text>
        <xsl:value-of select="position()"/>
      </item>
    <xsl:text>
</xsl:text>
    </xsl:for-each>
  </out>
</xsl:template>

</xsl:stylesheet>