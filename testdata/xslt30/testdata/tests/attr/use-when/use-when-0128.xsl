<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">


  <!-- PURPOSE: test use-when expression calling doc() function (should fail). -->
  <?spec xslt#conditional-inclusion?>
  <?error FODC0005?>

  <xsl:template match="*">
    <xsl:copy>
    <xsl:copy-of select="@*"/>
    <xsl:apply-templates/>
    </xsl:copy>
  </xsl:template>
  
  <xsl:template match="para" use-when="doc('usewhen905err.xsl')">
    <p><xsl:next-match/></p>
  </xsl:template>
  
</xsl:stylesheet>