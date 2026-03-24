<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">


  <!-- PURPOSE: test xsl:use-when on a literal result element -->
  <?spec xslt#conditional-inclusion?>

  <xsl:template match="*">
    <xsl:copy>
    <xsl:copy-of select="@*"/>
    <xsl:apply-templates/>
    </xsl:copy>
  </xsl:template>
  
  <xsl:template match="para">
    <pa><p xsl:use-when="contains(system-property('xsl:vendor-url'), 'http')"><xsl:next-match/></p></pa>
  </xsl:template>

</xsl:stylesheet>