<?xml version="1.0" encoding="iso-8859-1" ?>
<?spec xslt#sorting?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">


  <!-- Purpose: Test the famous technique for choosing the sort key dynamically. -->

  <xsl:param name="sortcol" select="'left'"/>
  <!-- Set value to either 'left' or 'right' above. Could change to external param. -->

  <xsl:template match="/">
    <out>
      <xsl:for-each select="doc/alpha">
        <xsl:sort select="./*[name(.) = $sortcol]" data-type="text"/>
        <z>
          <xsl:value-of select="concat(left,right,orig)"/>
          <xsl:text>,</xsl:text>
        </z>
      </xsl:for-each>
    </out>
  </xsl:template>

</xsl:stylesheet>
