<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">


  <!-- PURPOSE: test that invalid XSL code is allowed in an excluded section. -->
  <?spec xslt#conditional-inclusion?>

  <xsl:template match="*">
    <xsl:copy>
    <xsl:copy-of select="@*"/>
    <xsl:apply-templates/>
    </xsl:copy>
  </xsl:template>
  
  <xsl:template match="para" use-when="false()">
    <p><xsl:value-of my="word"/><xsl:next-matching-nonsense/></p>
  </xsl:template>

</xsl:stylesheet>