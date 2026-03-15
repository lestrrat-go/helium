<?xml version="1.0" encoding="iso-8859-1" ?>
<?spec xslt#sorting?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- Purpose: If nothing comes through select, should get document order. -->

  <xsl:template match="/">
    <out>
      <xsl:for-each select="doc/alpha">
        <xsl:sort select="./*[name(.) = 'never']" data-type="text"/>
        <z>
          <xsl:value-of select="concat(left,right,orig)"/>
          <xsl:text>,</xsl:text>
        </z>
      </xsl:for-each>
    </out>
  </xsl:template>

</xsl:stylesheet>
