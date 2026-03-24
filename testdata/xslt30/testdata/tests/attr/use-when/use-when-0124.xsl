<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">


  <!-- PURPOSE: ERROR: test use-when attempting to access global variable. -->
  <?spec xslt#conditional-inclusion?>
  <?error XPST0003?>  
  
  <xsl:param name="p" select="2"/>

  <xsl:template match="*">
    <xsl:copy>
    <xsl:copy-of select="@*"/>
    <xsl:apply-templates/>
    </xsl:copy>
  </xsl:template>
  
  <xsl:template match="para">
    <p><xsl:value-of select="." use-when="$p = 2"/></p>
  </xsl:template>

</xsl:stylesheet>