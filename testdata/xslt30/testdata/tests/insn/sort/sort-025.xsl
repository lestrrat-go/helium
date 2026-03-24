<?xml version="1.0" encoding="iso-8859-1" ?>
<?spec xslt#sorting?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- Purpose: Test for xsl:sort using a numeric expression for the select. -->

  <xsl:template match="/">
    <out>
      <xsl:for-each select="doc/alpha">
        <xsl:sort select="@height*@width" data-type="number"/>
        <xsl:value-of select="@net"/>
        <xsl:text>,</xsl:text>
      </xsl:for-each>
    </out>
  </xsl:template>

</xsl:stylesheet>
