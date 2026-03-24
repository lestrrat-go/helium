<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">


  <!-- PURPOSE: test use-when on xsl:include -->
  <?spec xslt#conditional-inclusion?>
  
  <xsl:include href="x-usewhen14inc.xsl" use-when="false()"/>

  <xsl:template match="*">
    <xsl:copy>
    <xsl:copy-of select="@*"/>
    <xsl:apply-templates/>
    </xsl:copy>
  </xsl:template>
  
  <xsl:template match="para">
    <p><xsl:next-match/></p>
  </xsl:template>

</xsl:stylesheet>