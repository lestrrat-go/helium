<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">


  <!-- PURPOSE: test use-when expression calling generate-id() function (should fail in 2.0, succeed in 3.0). -->
  <?spec xslt#conditional-inclusion?>
  <?error XPST0017?>

  <xsl:template match="*">
    <xsl:copy>
    <xsl:copy-of select="@*"/>
    <xsl:apply-templates/>
    </xsl:copy>
  </xsl:template>
  
  <xsl:template match="para" use-when="generate-id(()) = ''">
    <p><xsl:next-match/></p>
  </xsl:template>
  
</xsl:stylesheet>