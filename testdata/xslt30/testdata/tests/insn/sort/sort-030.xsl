<?xml version="1.0" encoding="iso-8859-1" ?>
<?spec xslt#sorting?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">


  <!-- Purpose: Test for xsl:sort using a numeric expression for the select. -->
  <!-- Note that we show that this sort does NOT do the expected rearrangement of nodes!
     Apparently, the conversion of the select expression to a string occurs at a bad time. -->

  <xsl:template match="/">
    <out>
      <xsl:for-each select="doc/alpha">
        <xsl:sort select="string-length(.)" data-type="number"/>
        <xsl:value-of select="@key"/>
        <xsl:text>|</xsl:text>
      </xsl:for-each>
    </out>
  </xsl:template>

</xsl:stylesheet>
