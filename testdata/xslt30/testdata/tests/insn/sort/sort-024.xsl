<?xml version="1.0" encoding="iso-8859-1" ?>
<?spec xslt#sorting?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">


  <!-- Purpose: Test for xsl:sort using a string function for the select. -->

  <xsl:template match="/">
    <out>
      <xsl:for-each select="doc/alpha">
        <xsl:sort select="substring(@key,2,1)" data-type="text"/>
        <xsl:value-of select="@key"/>
        <xsl:text>,</xsl:text>
      </xsl:for-each>
    </out>
  </xsl:template>

</xsl:stylesheet>
