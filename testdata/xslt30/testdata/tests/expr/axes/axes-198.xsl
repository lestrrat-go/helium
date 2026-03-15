<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#copying?>

  <xsl:template match="/">
  <out>
    <one><xsl:copy-of select="//c"/></one>
    <two><xsl:for-each select="//c"><xsl:copy/></xsl:for-each></two>
  </out>
  </xsl:template>
 
</xsl:stylesheet>
