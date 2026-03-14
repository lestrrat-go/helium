<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
     xmlns:a="http://a.uri/" xmlns:b="http://b.uri/">

<!-- test use of *:suffix pattern with strip-space -->
<?spec xslt#strip?>

<xsl:strip-space elements="doc *:a random"/>
  <xsl:template match="/">
    <o><xsl:copy-of select="doc/*"/></o>
  </xsl:template>
  
</xsl:stylesheet>
