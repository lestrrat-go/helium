<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">


  <!-- PURPOSE: test use-when="true()". -->
  <?spec xslt#conditional-inclusion?>

  <xsl:template match="*">
    <xsl:copy>
    <xsl:copy-of select="@*"/>
    <xsl:apply-templates/>
    </xsl:copy>
  </xsl:template>
  
  <xsl:template match="para" use-when="true()">
    <p><xsl:next-match/></p>
  </xsl:template>

</xsl:stylesheet>